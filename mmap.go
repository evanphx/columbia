package columbia

import (
	"github.com/evanphx/columbia/abi/linux"
	hclog "github.com/hashicorp/go-hclog"
)

func sysMmap(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		ptr  = args.Args.R0
		size = args.Args.R1
		// prot   = args.Args.R2
		flags = args.Args.R3
		// fd     = args.Args.R4
		// offset = args.Args.R5

		// fixed    = flags&linux.MAP_FIXED != 0
		private = flags&linux.MAP_PRIVATE != 0
		shared  = flags&linux.MAP_SHARED != 0
		anon    = flags&linux.MAP_ANONYMOUS != 0
		// map32bit = flags&linux.MAP_32BIT != 0
	)

	// Require exactly one of MAP_PRIVATE and MAP_SHARED.
	if private == shared {
		return -EINVAL
	}

	if anon {
		ptr = -1
	}

	reg, err := p.mem.NewRegion(ptr, size)
	if err != nil {
		return -EINVAL
	}

	l.Info("new region", "addr", reg.Start, "size", reg.Size)

	return reg.Start
}

func init() {
	syscalls[192] = sysMmap
}
