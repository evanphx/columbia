package columbia

import (
	"os"
	"reflect"

	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"
	hclog "github.com/hashicorp/go-hclog"
)

func NewLoader() *Loader {
	return &Loader{
		L: hclog.L(),
	}
}

type Loader struct {
	L hclog.Logger
}

func (l *Loader) LoadFile(path string) (*Module, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	m, err := wasm.ReadModule(f, l.importer)
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
	l.L.Info("importer", "name", name)
	if name == "env" {
		return l.envModule(), nil
	}

	return nil, nil
}

func (l *Loader) envModule() *wasm.Module {
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{},
			},
		},
	}

	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(l.syscall0),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(l.syscall1),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(l.syscall2),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[3],
			Host: reflect.ValueOf(l.syscall3),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[4],
			Host: reflect.ValueOf(l.syscall4),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[5],
			Host: reflect.ValueOf(l.syscall5),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(l.setjmp),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[6],
			Host: reflect.ValueOf(l.longjmp),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[7],
			Host: reflect.ValueOf(l.syscall6),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(l.syscall),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[8],
			Host: reflect.ValueOf(l.debug),
			Body: &wasm.FunctionBody{},
		},
	}

	m.Export = &wasm.SectionExports{
		Entries: map[string]wasm.ExportEntry{
			"__syscall": {
				FieldStr: "__syscall",
				Kind:     wasm.ExternalFunction,
				Index:    9,
			},
			"__syscall0": {
				FieldStr: "__syscall0",
				Kind:     wasm.ExternalFunction,
				Index:    0,
			},
			"__syscall1": {
				FieldStr: "__syscall1",
				Kind:     wasm.ExternalFunction,
				Index:    1,
			},
			"__syscall2": {
				FieldStr: "__syscall2",
				Kind:     wasm.ExternalFunction,
				Index:    2,
			},
			"__syscall3": {
				FieldStr: "__syscall3",
				Kind:     wasm.ExternalFunction,
				Index:    3,
			},
			"__syscall4": {
				FieldStr: "__syscall4",
				Kind:     wasm.ExternalFunction,
				Index:    4,
			},
			"__syscall5": {
				FieldStr: "__syscall5",
				Kind:     wasm.ExternalFunction,
				Index:    5,
			},
			"setjmp": {
				FieldStr: "setjmp",
				Kind:     wasm.ExternalFunction,
				Index:    6,
			},
			"longjmp": {
				FieldStr: "longjmp",
				Kind:     wasm.ExternalFunction,
				Index:    7,
			},
			"__syscall6": {
				FieldStr: "__syscall6",
				Kind:     wasm.ExternalFunction,
				Index:    8,
			},
			"debug": {
				FieldStr: "debug",
				Kind:     wasm.ExternalFunction,
				Index:    10,
			},
		},
	}

	return m
}
