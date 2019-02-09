package syscalls

import (
	"context"

	"github.com/evanphx/columbia/kernel"
	hclog "github.com/hashicorp/go-hclog"
)

func sysGetUID32(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return 0
}

func sysGetGID32(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return 0
}

func sysSetGID32(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return 0
}

func init() {
	Syscalls[199] = sysGetUID32
	Syscalls[200] = sysGetGID32
	Syscalls[214] = sysSetGID32
}
