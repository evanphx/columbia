package log

import hclog "github.com/hashicorp/go-hclog"

func EnableDebug() {
	L.SetLevel(hclog.Trace)
}
