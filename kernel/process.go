package kernel

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/evanphx/columbia/abi/linux"
	"github.com/evanphx/columbia/exec"
	"github.com/evanphx/columbia/fs"
	"github.com/evanphx/columbia/fs/tarfs"
	"github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/memory"
	"github.com/evanphx/columbia/pkg/ilist"
)

var (
	ErrUnknownFile = errors.New("unknown file")
)

type prockey struct{}

func GetTask(ctx context.Context) (*Task, bool) {
	if v := ctx.Value(prockey{}); v != nil {
		return v.(*Task), true
	}

	return nil, false
}

func SetTask(ctx context.Context, t *Task) context.Context {
	return context.WithValue(ctx, prockey{}, t)
}

type Task struct {
	*Process
}

func (t *Task) IP() int {
	return t.Process.Vm.IP()
}

type ProcessStatus int

const (
	Init    ProcessStatus = 0
	Running ProcessStatus = 1
	Dead    ProcessStatus = 2
)

type ExitStatus struct {
	Code  int
	Signo int
}

func (e ExitStatus) Status() int32 {
	return ((int32(e.Code) & 0xff) << 8) | (int32(e.Signo) & 0xff)
}

type Process struct {
	*exec.Process

	parent *Process
	pg     *ProcessGroup

	// Used by pg to implement Processes in the group as a list. Protected by
	// pg's mu.
	ilist.Entry

	Kernel     *Kernel
	Pid        int
	Mount      *fs.MountNamespace
	Mem        *memory.VirtualMemory
	Vm         *exec.VM
	EntryIndex int64

	status     ProcessStatus
	exitStatus ExitStatus
	fds        []*File

	waiters []chan int

	signals       Signals
	interruptFunc func()

	mu sync.Mutex
}

func (p *Process) PrintStack() {
	stack := p.Vm.Backtrace()
	os.Stderr.Write(stack)
}

func (p *Process) SetupTar(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer f.Close()

	tf, err := tarfs.NewTarFS(f)
	if err != nil {
		return err
	}

	root, err := tf.Root()
	if err != nil {
		return err
	}

	p.Mount = fs.NewMountNamespace()
	p.Mount.Root = &fs.Dirent{
		Inode: root,
	}

	return nil
}

func (p *Process) ReadCString(ptr int32) ([]byte, error) {
	var buf bytes.Buffer

	var t [1]byte

	off := int64(ptr)

	for {
		_, err := p.ReadAt(t[:], off)
		if err != nil {
			return nil, err
		}

		if t[0] == 0 {
			break
		}

		buf.WriteByte(t[0])
		off += 1
	}

	return buf.Bytes(), nil
}

func (p *Process) CopyOut(addr int32, val interface{}) error {
	return binary.Write(writeAdapter{sub: p, offset: int64(addr)}, binary.LittleEndian, val)
}

type readAdapter struct {
	sub    io.ReaderAt
	offset int64
}

func (ra readAdapter) Read(b []byte) (int, error) {
	return ra.sub.ReadAt(b, ra.offset)
}

func (p *Process) CopyIn(addr int32, val interface{}) error {
	return binary.Read(readAdapter{sub: p, offset: int64(addr)}, binary.LittleEndian, val)
}

func (p *Process) HookupStdio(i io.ReadCloser, o, e io.WriteCloser) {
	p.fds = append(p.fds, &File{
		refs: 1,
		r:    i,
	})

	p.fds = append(p.fds, &File{
		refs: 1,
		w:    o,
	})

	p.fds = append(p.fds, &File{
		refs: 1,
		w:    e,
	})
}

func (p *Process) CreatePipe() (*File, int, *File, int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pread, pwrite := io.Pipe()

	rfd := len(p.fds)

	read := &File{
		refs: 1,
		r:    pread,
	}

	write := &File{
		refs: 1,
		w:    pwrite,
	}

	p.fds = append(p.fds, read, write)

	return read, rfd, write, rfd + 1, nil
}

func (p *Process) Fork() (*Process, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	child := &Process{
		Kernel: p.Kernel,
		parent: p,
		pg:     p.pg,
	}

	p.Kernel.processes.AssignPid(child)

	child.pg.Add(child)

	child.Mem = p.Mem.Fork()

	for _, file := range p.fds {
		file.incRef()
		child.fds = append(child.fds, file)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, prockey{}, &Task{child})

	child.Mount = p.Mount
	child.Vm = p.Vm.Fork(ctx, child.Mem)
	child.Process = exec.NewProcess(child.Vm)

	child.Vm.Pid = p.Pid
	return child, nil
}

func (p *Process) Restart(args ...uint64) {
	p.Vm.Restart(args...)
}

func (p *Process) GetFile(fd int) (*File, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if fd >= len(p.fds) {
		return nil, false
	}

	file := p.fds[fd]
	if file == nil {
		return nil, false
	}

	return file, true
}

func (p *Process) CloseFile(fd int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if fd >= len(p.fds) {
		return ErrUnknownFile
	}

	file := p.fds[fd]
	if file == nil {
		return ErrUnknownFile
	}

	p.fds[fd] = nil

	return file.Close()
}

func (p *Process) Dup2(from, to int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	f := p.fds[to]
	if f != nil {
		f.Close()
	}

	p.fds[to] = p.fds[from]

	p.fds[to].incRef()

	return nil
}

/*
func (p *Process) WaitOn(ctx context.Context, target *Process, dur time.Duration) (int32, bool, error) {
	target.mu.Lock()
	if target.status == Dead {
		target.mu.Unlock()
		return target.exitStatus.Status(), true, nil
	}
	c := make(chan int, 1)
	target.waiters = append(target.waiters, c)
	target.mu.Unlock()

	timer := time.NewTimer(dur)

	defer timer.Stop()

	select {
	case code := <-c:
		return code, true, nil
	case <-ctx.Done():
		return 0, false, ctx.Err()
	case <-timer.C:
		return 0, false, nil
	}
}
*/

func (p *Process) WaitAnyChild(ctx context.Context, block bool) (int, ExitStatus, error) {
	target, err := p.pg.ReapAny(ctx, block)
	if err != nil {
		return 0, ExitStatus{}, err
	}

	if target == nil {
		return 0, ExitStatus{}, err
	}

	return target.Pid, target.exitStatus, nil
}

func (p *Process) Exit(code int) {
	log.L.Trace("process-exit", "pid", p.Pid, "code", code)

	for _, file := range p.fds {
		if file != nil {
			file.Close()
		}
	}

	p.mu.Lock()

	p.exitStatus.Code = code
	p.status = Dead

	p.mu.Unlock()

	p.pg.ProcessExitted(p)

	if p.Process != nil {
		p.Terminate()
	}

	p.parent.DeliverSignal(int(linux.SIGCHLD))
}

func (p *Process) Interrupt() {
	if p.interruptFunc != nil {
		p.interruptFunc()
	}
}

func (p *Process) SetInterrupt(f func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.interruptFunc = f
}

type ProcessManager struct {
	mu        sync.RWMutex
	highWater int
	processes map[int]*Process
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[int]*Process),
	}
}

func (p *ProcessManager) AssignPid(proc *Process) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 1; i <= p.highWater; i++ {
		if _, ok := p.processes[i]; !ok {
			proc.Pid = i
			p.processes[i] = proc
			return i
		}
	}

	p.highWater++
	pid := p.highWater
	p.processes[pid] = proc
	proc.Pid = pid

	return pid
}

func (p *ProcessManager) RemoveProc(proc *Process) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.processes, proc.Pid)
}
