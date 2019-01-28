package columbia

import (
	"encoding/binary"
	"time"

	hclog "github.com/hashicorp/go-hclog"
)

type timespec struct {
	Sec  int64
	NSec int32
}

var start = time.Now()

func sysClockGetTime(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		clk = args.Args.R0
		ptr = args.Args.R1
	)

	t := time.Now()

	var ts timespec

	switch clk {
	case 0:
		ts = timespec{
			Sec:  int64(t.Unix()),
			NSec: int32(t.Nanosecond()),
		}
	case 1, 6:
		diff := time.Since(start)
		ns := diff.Nanoseconds()
		ts = timespec{
			Sec:  ns / 1000000000,
			NSec: int32(ns % 1000000000),
		}
	default:
		return -EINVAL
	}

	err := binary.Write(writeAdapter{sub: p, offset: int64(ptr)}, binary.LittleEndian, ts)
	if err != nil {
		return -1
	}

	return 0
}

func init() {
	syscalls[265] = sysClockGetTime
}
