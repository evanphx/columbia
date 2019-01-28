package columbia

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
)

func sysOpen(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		ptr  = args.Args.R0
		mode = args.Args.R1
	)

	path, err := p.ReadCString(ptr)
	if err != nil {
		l.Error("error reading cstring", "error", err)
		return -1
	}

	fmt.Printf("open: %s, mode: %x\n", path, mode)
	return -1
}

func init() {
	syscalls[5] = sysOpen
}
