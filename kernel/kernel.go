package kernel

import "github.com/evanphx/columbia/wasm"

type Kernel struct {
	env *wasm.Module

	processes *ProcessManager
}

func NewKernel(env *wasm.Module) (*Kernel, error) {
	k := &Kernel{
		env:       env,
		processes: NewProcessManager(),
	}

	return k, nil
}

func (k *Kernel) EnvModule() *wasm.Module {
	return k.env
}
