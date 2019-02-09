package syscalls

import (
	"context"
	"encoding/binary"
	"io"
	"os"

	"golang.org/x/sys/unix"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/kernel"
	"github.com/evanphx/columbia/log"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
)

func sysClose(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		fd = args.Args.R0
	)

	err := task.CloseFile(int(fd))
	if err != nil {
		if errors.Cause(err) == kernel.ErrUnknownFile {
			return -abi.EINVAL
		}

		l.Error("error closing fd", "error", err, "fd", fd)
		return -abi.ENOSYS
	}

	return 0
}

func sysWrite(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		fd  = args.Args.R0
		ptr = args.Args.R1
		sz  = args.Args.R2
	)

	f, ok := task.GetFile(int(fd))
	if !ok {
		return -abi.EINVAL
	}

	w, ok := f.Writer()
	if !ok {
		return -abi.EBADF
	}

	data := make([]byte, sz)

	_, err := task.ReadAt(data, int64(ptr))
	if err != nil {
		log.L.Error("error reading data from userspace", "error", err)
		return -abi.EFAULT
	}

	n, err := w.Write(data)
	if err != nil {
		log.L.Error("error writing data", "error", err)
		return -abi.EFAULT
	}

	// log.L.Debug("write-data", "pid", task.Pid, "fd", fd, "data", spew.Sdump(data))

	return int32(n)
}

func sysWritev(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		fd  = args.Args.R0
		iov = args.Args.R1
		cnt = args.Args.R2
	)

	f, ok := task.GetFile(int(fd))
	if !ok {
		return -abi.EINVAL
	}

	w, ok := f.Writer()
	if !ok {
		return -abi.EBADF
	}

	tmp := make([]byte, 8)

	var ret int32

	for i := int32(0); i < cnt; i++ {
		_, err := task.ReadAt(tmp, int64(iov+(i*8)))
		if err != nil {
			return -1
		}

		ptr := binary.LittleEndian.Uint32(tmp)
		sz := binary.LittleEndian.Uint32(tmp[4:])

		// l.Info("read iov", "ptr", ptr, "sz", sz)

		data := make([]byte, sz)

		x, err := task.ReadAt(data, int64(ptr))
		if err != nil {
			return -1
		}

		ret += int32(x)

		// log.L.Debug("write-data", "pid", task.Pid, "fd", fd, "data", spew.Sdump(data))

		w.Write(data)
	}

	return ret
}

func sysRead(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		fd  = args.Args.R0
		buf = args.Args.R1
		sz  = args.Args.R2
	)

	f, ok := task.GetFile(int(fd))
	if !ok {
		return -abi.EINVAL
	}

	r, ok := f.Reader()
	if !ok {
		return -abi.EBADF
	}

	tmp := make([]byte, sz)

	n, err := r.Read(tmp)
	if err != nil {
		if err == io.EOF {
			return 0
		}

		if n == 0 || err != io.ErrUnexpectedEOF {
			l.Error("error reading", "error", err, "fd", fd)
			return -abi.EIO
		}
	}

	// log.L.Debug("read-data", "pid", task.Pid, "fd", fd, "data", spew.Sdump(tmp[:n]))

	err = task.CopyOut(buf, tmp[:n])
	if err != nil {
		l.Error("error copying data out", "error", err)
		return -abi.EFAULT
	}

	return int32(n)
}

func sysDup2(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		from = args.Args.R0
		to   = args.Args.R1
	)

	err := task.Dup2(int(from), int(to))
	if err != nil {
		l.Error("error duping fd", "from", from, "to", to)
		return -abi.EINVAL
	}

	return 0
}

func sysPipe(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var addr = args.Args.R0

	_, rfd, _, wfd, err := p.CreatePipe()
	if err != nil {
		l.Error("unable to create pipe", "error", err)
		return -kernel.ENOSYS
	}

	type pipeBuf struct {
		Read, Write int32
	}

	err = p.CopyOut(addr, pipeBuf{
		Read:  int32(rfd),
		Write: int32(wfd),
	})

	if err != nil {
		l.Error("error writing data to pipe buffer", "error", err)
		return -kernel.ENOSYS
	}

	return 0
}

func sysIOCTL(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		fd   = args.Args.R0
		cmd  = args.Args.R1
		addr = args.Args.R2
	)

	file, ok := p.GetFile(int(fd))
	if !ok {
		return -abi.EBADF
	}

	switch cmd {
	case 21523:
		var io *os.File

		r, ok := file.Reader()
		if ok {
			io, _ = r.(*os.File)
		}

		if io == nil {
			w, ok := file.Writer()
			if !ok {
				l.Error("not a writer")
				return -abi.EINVAL
			}

			io, _ = w.(*os.File)
		}

		if io == nil {
			return -abi.EINVAL
		}

		ws, err := unix.IoctlGetWinsize(int(io.Fd()), unix.TIOCGWINSZ)
		if err != nil {
			return -abi.EINVAL
		}

		err = p.CopyOut(addr, ws)
		if err != nil {
			l.Error("error copying data to userspace", "error", err)
			return -kernel.ENOSYS
		}

		return 0
	default:
		return -abi.EINVAL
	}
}

func init() {
	Syscalls[6] = sysClose

	Syscalls[4] = sysWrite
	Syscalls[146] = sysWritev

	Syscalls[3] = sysRead

	Syscalls[42] = sysPipe

	Syscalls[63] = sysDup2
	Syscalls[54] = sysIOCTL
}
