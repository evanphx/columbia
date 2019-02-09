package syscalls

import (
	"context"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/kernel"
	hclog "github.com/hashicorp/go-hclog"
)

func sysRtSigProcMask(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return 0
}

type kSigAction struct {
	Handler int32
	Flags   int32
}

func sysRtSigaction(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		signo      = args.Args.R0
		actionAddr = args.Args.R1
	)

	var act kSigAction

	err := p.CopyIn(actionAddr, &act)
	if err != nil {
		l.Error("error copying sigaction", "error", err)
		return -abi.EFAULT
	}

	p.AddSignalHandler(int(signo), int64(act.Handler))

	return 0
}

func init() {
	Syscalls[174] = sysRtSigaction
}
