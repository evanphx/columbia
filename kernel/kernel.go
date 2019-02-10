package kernel

import (
	"github.com/evanphx/columbia/loader"
	"github.com/evanphx/columbia/wasm"
)

type Kernel struct {
	env         *wasm.Module
	loaderCache *loader.LoaderCache

	processes *ProcessManager
}

func NewKernel(env *wasm.Module) (*Kernel, error) {
	k := &Kernel{
		env:         env,
		loaderCache: loader.NewLoaderCache(),
		processes:   NewProcessManager(),
	}

	return k, nil
}

func (k *Kernel) EnvModule() *wasm.Module {
	return k.env
}
