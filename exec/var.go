// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

func (vm *VM) getLocal() {
	index := vm.fetchUint32()
	vm.pushUint64(vm.stack[vm.frame.fp+int64(index)])
}

func (vm *VM) setLocal() {
	index := vm.fetchUint32()
	vm.stack[vm.frame.fp+int64(index)] = vm.popUint64()
}

func (vm *VM) teeLocal() {
	index := vm.fetchUint32()
	val := vm.stack[vm.frame.sp]
	vm.stack[vm.frame.fp+int64(index)] = val
}

func (vm *VM) getGlobal() {
	index := vm.fetchUint32()
	vm.pushUint64(vm.globals[int(index)])
}

func (vm *VM) setGlobal() {
	index := vm.fetchUint32()
	vm.globals[int(index)] = vm.popUint64()
}
