package columbia

import (
	"context"
	"encoding/binary"

	"github.com/evanphx/columbia/exec"
)

func (l *Loader) setjmp(ctx context.Context, arg int32) int32 {
	l.L.Info("setjmp", "addr", arg)
	p := ctx.Value(prockey{}).(*Process)

	buf := p.GetContext()

	err := binary.Write(writeAdapter{sub: p, offset: int64(arg)}, binary.LittleEndian, buf)
	if err != nil {
		l.L.Error("error writing jmpbuf", "error", err)
		return -EINVAL
	}

	return 0
}

func (l *Loader) longjmp(ctx context.Context, addr, val int32) {
	l.L.Info("longjmp", "addr", addr, "val", val)

	p := ctx.Value(prockey{}).(*Process)

	var buf exec.JmpBuf

	err := binary.Read(readAdapter{sub: p, offset: int64(addr)}, binary.LittleEndian, &buf)
	if err != nil {
		l.L.Error("error writing jmpbuf", "error", err)
		return
	}

	p.SetContext(&buf, uint64(val))
}
