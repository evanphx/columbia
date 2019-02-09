package syscalls

import (
	"context"

	"github.com/evanphx/columbia/kernel"
	hclog "github.com/hashicorp/go-hclog"
)

type SysArgs struct {
	Index int32
	Args  SyscallRequest
}

type SyscallRequest struct {
	R0, R1, R2, R3, R4, R5, R6 int32
}

type Process interface{}

var Syscalls [1024]func(context.Context, hclog.Logger, *kernel.Task, SysArgs) int32
