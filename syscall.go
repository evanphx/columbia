package columbia

import (
	"context"
	"encoding/binary"
	"io"

	hclog "github.com/hashicorp/go-hclog"
)

type sysArgs struct {
	Index int32
	Args  syscallRequest
}

type syscallRequest struct {
	R0, R1, R2, R3, R4, R5, R6 int32
}

func (l *Loader) invokeSyscall(ctx context.Context, args sysArgs) int32 {
	if f := syscalls[args.Index]; f != nil {
		p := ctx.Value(prockey{}).(*Process)
		return f(l.L, p, args)
	}

	return -1
}

func (l *Loader) syscall0(ctx context.Context, idx int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx])

	return l.invokeSyscall(ctx, sysArgs{Index: idx})
}

func (l *Loader) syscall1(ctx context.Context, idx, a int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a)
	if idx == 1 || idx == 252 {
		p := ctx.Value(prockey{}).(*Process)
		p.Terminate()
	}
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a}})
}

func (l *Loader) syscall2(ctx context.Context, idx, a, b int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a, "b", b)
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a, R1: b}})
}

func (l *Loader) syscall3(ctx context.Context, idx, a, b, c int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a, "b", b, "c", c)
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a, R1: b, R2: c}})
}

func (l *Loader) syscall4(ctx context.Context, idx, a, b, c, d int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a, "b", b, "c", c, "d", d)
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a, R1: b, R2: c, R3: d}})
}

func (l *Loader) syscall5(ctx context.Context, idx, a, b, c, d, e int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a, "b", b, "c", c, "d", d, "e", e)
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a, R1: b, R2: c, R3: d, R4: e}})
}

func (l *Loader) syscall6(ctx context.Context, idx, a, b, c, d, e, f int32) int32 {
	l.L.Info("syscall", "index", idx, "name", SyscallNames[idx], "a", a, "b", b, "c", c, "d", d, "e", e, "f", f)
	return l.invokeSyscall(ctx, sysArgs{Index: idx, Args: syscallRequest{R0: a, R1: b, R2: c, R3: d, R4: e, R5: f}})
}

type readAdapter struct {
	sub    io.ReaderAt
	offset int64
}

func (ra readAdapter) Read(b []byte) (int, error) {
	return ra.sub.ReadAt(b, ra.offset)
}

func (l *Loader) syscall(ctx context.Context, idx, addr int32) int32 {
	var args sysArgs

	args.Index = idx

	p := ctx.Value(prockey{}).(*Process)

	err := binary.Read(readAdapter{sub: p, offset: int64(addr)}, binary.LittleEndian, &args.Args)
	if err != nil {
		l.L.Error("error decoding syscall", "error", err)
		return -1
	}

	l.L.Info("syscall-vararg", "index", idx, "name", SyscallNames[idx], "req", args.Args)

	return l.invokeSyscall(ctx, args)
}

var syscalls [1024]func(hclog.Logger, *Process, sysArgs) int32
