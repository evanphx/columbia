package loader

import (
	"io"
	"os"

	"github.com/evanphx/columbia/wasm"
	"github.com/evanphx/columbia/wasm/validate"
	hclog "github.com/hashicorp/go-hclog"
)

func NewLoader() *Loader {
	return &Loader{
		L: hclog.L(),
	}
}

type Loader struct {
	L   hclog.Logger
	env *wasm.Module
}

func (l *Loader) LoadFile(path string, env *wasm.Module) (*Module, error) {
	l.env = env

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return l.Load(f, env)
}

func (l *Loader) Load(r io.Reader, env *wasm.Module) (*Module, error) {
	l.env = env

	m, err := wasm.ReadModule(r, l.importer)
	if err != nil {
		return nil, err
	}

	err = validate.VerifyModule(m)
	if err != nil {
		return nil, err
	}

	return &Module{l, m}, nil
}

func (l *Loader) importer(name string) (*wasm.Module, error) {
	if name == "env" {
		return l.env, nil
	}

	return nil, nil
}
