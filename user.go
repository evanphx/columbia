package columbia

import (
	hclog "github.com/hashicorp/go-hclog"
)

func sysGetUID32(l hclog.Logger, p *Process, args sysArgs) int32 {
	return 0
}

func sysGetGID32(l hclog.Logger, p *Process, args sysArgs) int32 {
	return 0
}

func sysSetGID32(l hclog.Logger, p *Process, args sysArgs) int32 {
	return 0
}

func init() {
	syscalls[199] = sysGetUID32
	syscalls[200] = sysGetGID32
	syscalls[214] = sysSetGID32
}
