package loader

import (
	"github.com/go-interpreter/wagon/wasm"
)

type Module struct {
	loader *Loader
	Module *wasm.Module
}
