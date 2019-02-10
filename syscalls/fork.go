package syscalls

import (
	"context"
	"time"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/kernel"
	"github.com/evanphx/columbia/log"
	hclog "github.com/hashicorp/go-hclog"
)

func sysFork(ctx context.Context, l hclog.Logger, p *kernel.Task, arg SysArgs) int32 {
	child, err := p.Fork()
	if err != nil {
		l.Error("error forking process", "error", err)
		return -kernel.ENOSYS
	}

	go child.Restart(0)

	return int32(child.Pid)
}

const (
	WNOHANG = 1
)

func sysWait4(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		pid      = args.Args.R0
		statAddr = args.Args.R1
		flags    = args.Args.R2
	)

	switch pid {
	case -1:
		pid, status, err := p.WaitAnyChild(ctx, flags&WNOHANG == 0)
		if err != nil {
			if err == context.Canceled {
				return -abi.EINTR
			}

			l.Error("error waiting for any child process", "error", err)
			return -abi.ENOSYS
		}

		if pid == 0 {
			log.L.Trace("wait4-no-child")
			return -abi.ECHILD
		}

		p.CopyOut(statAddr, status.Status())

		log.L.Trace("wait4-found-child", "pid", pid, "status", status.Code)
		return int32(pid)
	default:
		time.Sleep(1 * time.Second)
		return -1
	}
}

func init() {
	Syscalls[2] = sysFork
	Syscalls[114] = sysWait4
}
