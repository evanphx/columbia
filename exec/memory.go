// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"errors"
	"math"

	hclog "github.com/hashicorp/go-hclog"
)

type Memory interface {
	Project(offset, size int32) ([]byte, error)
	Size() int
	Grow(size int32) error
}

var ErrInvalidMemoryAccess = errors.New("invalid memory access")

type SliceMemory struct {
	mem []byte
}

func (s *SliceMemory) Memory() []byte {
	return s.mem
}

func (s *SliceMemory) Size() int {
	return len(s.mem)
}

func (s *SliceMemory) Project(offset, size int32) ([]byte, error) {
	if int(offset+size) > len(s.mem) {
		return nil, ErrInvalidMemoryAccess
	}

	if offset >= 0x30000-1000 {
		hclog.L().Info("slice project", "offset", offset, "size", size)
	}

	return s.mem[offset : offset+size], nil
}

func (s *SliceMemory) Grow(additional int32) error {
	s.mem = append(s.mem, make([]byte, additional)...)
	return nil
}

func NewSliceMemory(b []byte) *SliceMemory {
	return &SliceMemory{mem: b}
}

// ErrOutOfBoundsMemoryAccess is the error value used while trapping the VM
// when it detects an out of bounds access to the linear memory.
var ErrOutOfBoundsMemoryAccess = errors.New("exec: out of bounds memory access")

func (vm *VM) fetchBaseAddr() int {
	return int(vm.fetchUint32() + uint32(vm.popInt32()))
}

// curMem returns a slice to the memeory segment pointed to by
// the current base address on the bytecode stream.
func (vm *VM) curMem(sz int32) []byte {
	slice, err := vm.memory.Project(int32(vm.fetchBaseAddr()), sz)
	if err != nil {
		panic(err)
	}

	return slice
}

func (vm *VM) i32Load() {
	vm.pushUint32(endianess.Uint32(vm.curMem(4)))
}

func (vm *VM) i32Load8s() {
	vm.pushInt32(int32(int8(vm.curMem(1)[0])))
}

func (vm *VM) i32Load8u() {
	vm.pushUint32(uint32(uint8(vm.curMem(1)[0])))
}

func (vm *VM) i32Load16s() {
	vm.pushInt32(int32(int16(endianess.Uint16(vm.curMem(2)))))
}

func (vm *VM) i32Load16u() {
	vm.pushUint32(uint32(endianess.Uint16(vm.curMem(2))))
}

func (vm *VM) i64Load() {
	vm.pushUint64(endianess.Uint64(vm.curMem(8)))
}

func (vm *VM) i64Load8s() {
	vm.pushInt64(int64(int8(vm.curMem(1)[0])))
}

func (vm *VM) i64Load8u() {
	vm.pushUint64(uint64(uint8(vm.curMem(1)[0])))
}

func (vm *VM) i64Load16s() {
	vm.pushInt64(int64(int16(endianess.Uint16(vm.curMem(2)))))
}

func (vm *VM) i64Load16u() {
	vm.pushUint64(uint64(endianess.Uint16(vm.curMem(2))))
}

func (vm *VM) i64Load32s() {
	vm.pushInt64(int64(int32(endianess.Uint32(vm.curMem(4)))))
}

func (vm *VM) i64Load32u() {
	vm.pushUint64(uint64(endianess.Uint32(vm.curMem(8))))
}

func (vm *VM) f32Store() {
	v := math.Float32bits(vm.popFloat32())
	endianess.PutUint32(vm.curMem(4), v)
}

func (vm *VM) f32Load() {
	vm.pushFloat32(math.Float32frombits(endianess.Uint32(vm.curMem(4))))
}

func (vm *VM) f64Store() {
	v := math.Float64bits(vm.popFloat64())
	endianess.PutUint64(vm.curMem(8), v)
}

func (vm *VM) f64Load() {
	vm.pushFloat64(math.Float64frombits(endianess.Uint64(vm.curMem(8))))
}

func (vm *VM) i32Store() {
	v := vm.popUint32()
	endianess.PutUint32(vm.curMem(4), v)
}

func (vm *VM) i32Store8() {
	v := byte(uint8(vm.popUint32()))
	vm.curMem(1)[0] = v
}

func (vm *VM) i32Store16() {
	v := uint16(vm.popUint32())
	endianess.PutUint16(vm.curMem(2), v)
}

func (vm *VM) i64Store() {
	v := vm.popUint64()
	endianess.PutUint64(vm.curMem(8), v)
}

func (vm *VM) i64Store8() {
	v := byte(uint8(vm.popUint64()))
	vm.curMem(1)[0] = v
}

func (vm *VM) i64Store16() {
	v := uint16(vm.popUint64())
	endianess.PutUint16(vm.curMem(2), v)
}

func (vm *VM) i64Store32() {
	v := uint32(vm.popUint64())
	endianess.PutUint32(vm.curMem(4), v)
}

func (vm *VM) currentMemory() {
	_ = vm.fetchInt8() // reserved (https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/BinaryEncoding.md#memory-related-operators-described-here)
	vm.pushInt32(int32(vm.memory.Size() / wasmPageSize))
}

func (vm *VM) growMemory() {
	_ = vm.fetchInt8() // reserved (https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/BinaryEncoding.md#memory-related-operators-described-here)
	curLen := vm.memory.Size() / wasmPageSize
	n := vm.popInt32()
	vm.memory.Grow(n * wasmPageSize)
	vm.pushInt32(int32(curLen))
}
