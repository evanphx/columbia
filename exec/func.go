// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	gcontext "context"
	"fmt"
	"math"
	"reflect"

	"github.com/evanphx/columbia/exec/internal/compile"
)

type function interface {
	call(vm *VM, index int64)
}

type compiledFunction struct {
	code           []byte
	file           string
	line           int
	branchTables   []*compile.BranchTable
	maxDepth       int  // maximum stack depth reached while executing the function body
	totalLocalVars int  // number of local variables used by the function
	args           int  // number of arguments the function accepts
	returns        bool // whether the function returns a value
	name           string
	offsets        map[int]int64
}

type goFunction struct {
	val reflect.Value
	typ reflect.Type
}

type vmctx struct{}

func setVM(ctx gcontext.Context, vm *VM) gcontext.Context {
	return gcontext.WithValue(ctx, vmctx{}, vm)
}

func GetProcess(ctx gcontext.Context) *Process {
	v := ctx.Value(vmctx{})
	if v == nil {
		panic("not a VM context")
	}

	return NewProcess(v.(*VM))
}

func (fn goFunction) call(vm *VM, index int64) {
	// numIn = # of call inputs + vm, as the function expects
	// an additional *VM argument
	numIn := fn.typ.NumIn()
	args := make([]reflect.Value, numIn)

	/*
		// Pass proc as an argument. Check that the function indeed
		// expects a *Process argument.
		if reflect.ValueOf(gcontext.Context).Kind() != fn.typ.In(0).Kind() {
			panic(fmt.Sprintf("exec: the first argument of a host function was %s, expected %s", fn.typ.In(0).Kind(), reflect.ValueOf(vm.gctx).Kind()))
		}
	*/
	args[0] = reflect.ValueOf(vm.gctx)

	for i := numIn - 1; i >= 1; i-- {
		val := reflect.New(fn.typ.In(i)).Elem()
		raw := vm.popUint64()
		kind := fn.typ.In(i).Kind()

		switch kind {
		case reflect.Float64, reflect.Float32:
			val.SetFloat(math.Float64frombits(raw))
		case reflect.Uint32, reflect.Uint64:
			val.SetUint(raw)
		case reflect.Int32, reflect.Int64:
			val.SetInt(int64(raw))
		default:
			panic(fmt.Sprintf("exec: args %d invalid kind=%v", i, kind))
		}

		args[i] = val
	}

	pre := vm.frame

	rtrns := fn.val.Call(args)

	// If the caller manipulated the frames, then presume it is going to handle
	// pushing the return value directly, so we ignore it.
	if vm.frame != pre {
		return
	}

	for i, out := range rtrns {
		kind := out.Kind()
		switch kind {
		case reflect.Float64, reflect.Float32:
			vm.pushFloat64(out.Float())
		case reflect.Uint32, reflect.Uint64:
			vm.pushUint64(out.Uint())
		case reflect.Int32, reflect.Int64:
			vm.pushInt64(out.Int())
		default:
			panic(fmt.Sprintf("exec: return value %d invalid kind=%v", i, kind))
		}
	}
}

func (compiled *compiledFunction) call(vm *VM, index int64) {
	callerFrame := vm.frame
	vm.frameIdx++
	nextFrame := &vm.frames[vm.frameIdx]

	// Overlap the next frame pointer with the callers stack that
	// containers the arguments as the locals (ie no copy)
	nextFrame.fp = callerFrame.sp - int64(compiled.args) + 1

	freeLocals := int64(compiled.totalLocalVars) - int64(compiled.args)

	nextFrame.sp = callerFrame.sp + freeLocals

	// Backup the callers sp now so when it's restored, the stack is correct.
	callerFrame.sp -= int64(compiled.args)

	/*
		if freeLocals > 0 {
			for i := nextFrame.fp + int64(compiled.args); i <= nextFrame.sp; i++ {
				vm.stack[i] = 0
			}
		}
	*/

	nextFrame.ip = 0
	nextFrame.fn = compiled
	nextFrame.code = compiled.code

	Debugf("|> call frame=%d sp=%d fp=%d args=%d local=%d\n", vm.frameIdx, nextFrame.sp,
		nextFrame.fp, compiled.args, compiled.totalLocalVars)

	// newStack := make([]uint64, compiled.maxDepth)
	// locals := make([]uint64, compiled.totalLocalVars)

	// for i := compiled.args - 1; i >= 0; i-- {
	// locals[i] = vm.popUint64()
	// }

	//save execution context

	vm.frame = nextFrame

	maxDepth := int(vm.frame.sp) + compiled.maxDepth

	if len(vm.stack) < maxDepth {
		add := 1024
		if maxDepth-len(vm.stack) > add {
			add = (maxDepth - len(vm.stack)) + 128
		}

		vm.stack = append(vm.stack, make([]uint64, add)...)
	}

	/*
		for i := 0; i < vm.frameIdx; i++ {
			fmt.Print("  ")
		}
		fmt.Printf("%s\n", vm.Location())
	*/
}
