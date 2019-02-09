package loader

import (
	"github.com/evanphx/columbia/wasm"
)

type Module struct {
	loader *Loader
	Module *wasm.Module
}
