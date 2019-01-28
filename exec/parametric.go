// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

func (vm *VM) drop() {
	vm.frame.sp--
}

func (vm *VM) selectOp() {
	c := vm.popUint32()
	val2 := vm.popUint64()
	val1 := vm.popUint64()

	if c != 0 {
		vm.pushUint64(val1)
	} else {
		vm.pushUint64(val2)
	}
}
