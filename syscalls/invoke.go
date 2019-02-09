package syscalls

import (
	"context"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/kernel"
	"github.com/evanphx/columbia/log"
)

type Invoker struct {
	Kernel *kernel.Kernel
}

func (i *Invoker) InvokeSyscall(ctx context.Context, args SysArgs) int32 {
	if f := Syscalls[args.Index]; f != nil {
		ctx, cancel := context.WithCancel(ctx)

		p, ok := kernel.GetTask(ctx)
		if !ok {
			return -abi.ENOSYS
		}

		p.SetInterrupt(cancel)

		ret := f(ctx, log.L, p, args)

		if p.CheckInterrupt() {
			return -abi.EINTR
		}

		return ret
	}

	return -1
}
