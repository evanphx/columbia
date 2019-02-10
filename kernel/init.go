package kernel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/evanphx/columbia/exec"
	"github.com/evanphx/columbia/loader"
	"github.com/evanphx/columbia/memory"
)

func (k *Kernel) StartProcess(proc *Process) error {
	_, err := proc.Vm.ExecCode(proc.EntryIndex)
	return err
}

var ErrNoStart = errors.New("no _start function defined")

func (k *Kernel) InitProcess(ctx context.Context, path string, args []string, env []string) (*Process, error) {
	proc := &Process{
		Kernel: k,
		pg:     &ProcessGroup{},
	}

	k.processes.AssignPid(proc)

	err := proc.SetupTar("tmp/test.tar")
	if err != nil {
		return nil, err
	}

	task := &Task{Process: proc}

	ctx = SetTask(ctx, task)

	return k.SetupProcess(ctx, proc, path, args, env)
}

func (k *Kernel) SetupProcess(ctx context.Context, proc *Process, path string, args []string, env []string) (*Process, error) {
	dirent, err := proc.Mount.LookupPath(ctx, path)
	if err != nil {
		return nil, err
	}

	r, err := dirent.Reader()
	if err != nil {
		return nil, err
	}

	l := loader.NewLoader(k.loaderCache)
	m, err := l.Load(r, k.env)
	if err != nil {
		return nil, err
	}

	virtmem := memory.NewVirtualMemory()
	_, err = virtmem.NewRegion(0, int32(m.Module.Memory.Entries[0].Limits.Initial)*memory.WasmPageSize)
	if err != nil {
		return nil, err
	}

	proc.Mem = virtmem

	vm, err := exec.NewVM(ctx, m.Module, virtmem)
	if err != nil {
		return nil, err
	}

	// Terminate the old VM so it exits
	if proc.Vm != nil {
		proc.Terminate()
	}

	proc.Vm = vm
	proc.Process = exec.NewProcess(vm)

	vm.Pid = proc.Pid

	entry, ok := m.Module.Export.Entries["_start"]
	if !ok {
		return nil, ErrNoStart
	}

	proc.EntryIndex = int64(entry.Index)

	ent, ok := m.Module.Export.Entries["__heap_base"]
	if !ok {
		return nil, fmt.Errorf("no __heap_base")
	}

	gbl := m.Module.GlobalIndexSpace[ent.Index]

	v, err := m.Module.ExecInitExpr(gbl.Init)
	if err != nil {
		return nil, err
	}

	ptr, ok := v.(int32)
	if !ok {
		return nil, fmt.Errorf("not a int32")
	}

	d0 := ptr - 16
	sca := d0 + 12

	writeExecHeader(sca, vm.Memory(), args, env)

	return proc, nil
}

func writeExecHeader(base int32, vmem exec.Memory, args []string, env []string) error {
	dataStart := 4 + // argc
		(4 * len(args)) + // argv
		4 + // null
		(4 * len(env)) + //envp
		4 + // null
		4 + // auxv
		4 // null

	total := dataStart

	for _, str := range args {
		total += len(str) + 1
	}

	for _, str := range env {
		total += len(str) + 1
	}

	mem, err := vmem.Project(base, int32(total))
	if err != nil {
		return err
	}

	le := binary.LittleEndian

	le.PutUint32(mem, uint32(len(args)))

	nextStr := int32(dataStart)

	ptr := mem[4:]
	for _, str := range args {
		le.PutUint32(ptr, uint32(base+nextStr))
		copy(mem[nextStr:], []byte(str))
		mem[nextStr+int32(len(str))] = 0
		nextStr += int32(len(str) + 1)
		ptr = ptr[4:]
	}
	le.PutUint32(ptr, 0) // null after argv
	ptr = ptr[4:]

	for _, str := range env {
		le.PutUint32(ptr, uint32(base+nextStr))
		copy(mem[nextStr:], []byte(str))
		mem[nextStr+int32(len(str))] = 0
		nextStr += int32(len(str) + 1)
		ptr = ptr[4:]
	}

	le.PutUint32(ptr, 0) // null after envp
	ptr = ptr[4:]
	le.PutUint32(ptr, 0) // auxv
	ptr = ptr[4:]
	le.PutUint32(ptr, 0) // null after auxv

	return nil
}
