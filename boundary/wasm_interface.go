package boundary

import (
	"context"
	"reflect"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/exec"
	"github.com/evanphx/columbia/kernel"
	"github.com/evanphx/columbia/syscalls"
	"github.com/evanphx/columbia/wasm"
	hclog "github.com/hashicorp/go-hclog"
)

type SyscallInvoker interface {
	InvokeSyscall(context.Context, syscalls.SysArgs) int32
}

type WasmInterface struct {
	L       hclog.Logger
	Invoker SyscallInvoker
}

func (w *WasmInterface) invokeSyscall(ctx context.Context, args syscalls.SysArgs) int32 {
	return w.Invoker.InvokeSyscall(ctx, args)
}

func (w *WasmInterface) syscall0(ctx context.Context, idx int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx])

	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx})
}

func (w *WasmInterface) syscall1(ctx context.Context, idx, a int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a}})
}

func (w *WasmInterface) syscall2(ctx context.Context, idx, a, b int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a, "b", b)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a, R1: b}})
}

func (w *WasmInterface) syscall3(ctx context.Context, idx, a, b, c int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a, "b", b, "c", c)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a, R1: b, R2: c}})
}

func (w *WasmInterface) syscall4(ctx context.Context, idx, a, b, c, d int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a, "b", b, "c", c, "d", d)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a, R1: b, R2: c, R3: d}})
}

func (w *WasmInterface) syscall5(ctx context.Context, idx, a, b, c, d, e int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a, "b", b, "c", c, "d", d, "e", e)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a, R1: b, R2: c, R3: d, R4: e}})
}

func (w *WasmInterface) syscall6(ctx context.Context, idx, a, b, c, d, e, f int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	w.L.Trace("syscall", "pid", p.Pid, "index", idx, "name", syscalls.SyscallNames[idx], "a", a, "b", b, "c", c, "d", d, "e", e, "f", f)
	return w.invokeSyscall(ctx, syscalls.SysArgs{Index: idx, Args: syscalls.SyscallRequest{R0: a, R1: b, R2: c, R3: d, R4: e, R5: f}})
}

func (w *WasmInterface) syscall(ctx context.Context, idx, addr int32) int32 {
	var args syscalls.SysArgs

	args.Index = idx

	p, ok := kernel.GetTask(ctx)
	if !ok {
		return -abi.ENOSYS
	}

	err := p.CopyIn(addr, &args.Args)
	if err != nil {
		w.L.Error("error decoding syscall", "error", err)
		return -1
	}

	w.L.Trace("syscall", "pid", p.Pid, "ip", p.Vm.IP(), "index", idx, "name", syscalls.SyscallNames[idx], "req", args.Args)

	return w.invokeSyscall(ctx, args)
}

func (w *WasmInterface) setjmp(ctx context.Context, arg int32) int32 {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		w.L.Error("unknown task context")
		return -abi.ENOSYS
	}

	err := p.CopyOut(arg, p.GetContext())
	if err != nil {
		w.L.Error("error writing jmpbuf", "error", err)
		return -abi.EINVAL
	}

	return 0
}

func (w *WasmInterface) longjmp(ctx context.Context, addr, val int32) {
	p, ok := kernel.GetTask(ctx)
	if !ok {
		w.L.Error("unknown task context")
		return
	}

	var buf exec.JmpBuf

	err := p.CopyIn(addr, &buf)
	if err != nil {
		w.L.Error("error writing jmpbuf", "error", err)
		return
	}

	p.SetContext(&buf, uint64(val))
}

func (w *WasmInterface) EnvModule() *wasm.Module {
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
			Host: reflect.ValueOf(w.syscall0),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(w.syscall1),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(w.syscall2),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[3],
			Host: reflect.ValueOf(w.syscall3),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[4],
			Host: reflect.ValueOf(w.syscall4),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[5],
			Host: reflect.ValueOf(w.syscall5),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(w.setjmp),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[6],
			Host: reflect.ValueOf(w.longjmp),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[7],
			Host: reflect.ValueOf(w.syscall6),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(w.syscall),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[8],
			Host: reflect.ValueOf(w.debug),
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
