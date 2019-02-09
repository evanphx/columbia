package log

import hclog "github.com/hashicorp/go-hclog"

var L hclog.Logger

func init() {
	L = hclog.New(&hclog.LoggerOptions{})
	L.SetLevel(hclog.Info)
}
