package log

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
)

var L hclog.Logger

func init() {
	L = hclog.New(&hclog.LoggerOptions{})
	L.SetLevel(hclog.Info)

	if str := os.Getenv("TRACE"); str != "" {
		L.SetLevel(hclog.Trace)
	}
}
