package log

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
)

func EnableDebug() {
	if str := os.Getenv("TRACE"); str != "" {
		L.SetLevel(hclog.Trace)
	}
}
