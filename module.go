package columbia

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/evanphx/columbia/exec"
	"github.com/go-interpreter/wagon/wasm"
)

type Module struct {
	loader *Loader
	module *wasm.Module
}

var ErrNoStart = errors.New("no _start function defined")

type prockey struct{}

func (m *Module) Run(args []string) error {
	proc := &Process{}

	ctx := context.Background()
	ctx = context.WithValue(ctx, prockey{}, proc)

	virtmem := NewVirtualMemory()
	_, err := virtmem.NewRegion(0, int32(m.module.Memory.Entries[0].Limits.Initial)*wasmPageSize)
	if err != nil {
		return err
	}

	err = proc.SetupTar("tmp/test.tar")
	if err != nil {
		return err
	}

	proc.mem = virtmem

	vm, err := exec.NewVM(ctx, m.module, virtmem)
	if err != nil {
		return err
	}

	proc.Process = exec.NewProcess(vm)

	// Switch to calling _start_c, which takes the pointer to the top of the stack
	entry, ok := m.module.Export.Entries["_start"]
	if !ok {
		return ErrNoStart
	}

	ent, ok := m.module.Export.Entries["__heap_base"]
	if !ok {
		return fmt.Errorf("no __heap_base")
	}

	gbl := m.module.GlobalIndexSpace[ent.Index]

	v, err := m.module.ExecInitExpr(gbl.Init)
	if err != nil {
		return err
	}

	ptr, ok := v.(int32)
	if !ok {
		return fmt.Errorf("not a int32")
	}

	d0 := ptr - 16
	sca := d0 + 12

	/*
		mem, err := vm.Memory().Project(sca, 50)
		if err != nil {
			return err
		}

			binary.LittleEndian.PutUint32(mem[:], 1)               // argc
			binary.LittleEndian.PutUint32(mem[4:], uint32(sca+28)) // argv
			binary.LittleEndian.PutUint32(mem[8:], 0)              // 0
			binary.LittleEndian.PutUint32(mem[12:], 0)             // envp
			binary.LittleEndian.PutUint32(mem[16:], 0)             // 0
			binary.LittleEndian.PutUint32(mem[20:], 0)             // auxv
			binary.LittleEndian.PutUint32(mem[24:], 0)             // 0
			copy(mem[28:], []byte("sh\u0000-c\u0000date\u0000"))
	*/

	writeExecHeader(sca, vm.Memory(), args, []string{"USER=root"})

	ret, err := vm.ExecCode(int64(entry.Index))
	if err != nil {
		return err
	}

	spew.Dump(ret)

	return nil
}

func writeExecHeader(base int32, vmem exec.Memory, args []string, env []string) error {
	dataStart := 4 + // argc
		(4 * len(args)) + // argv
		4 + // null
		(4 * len(env)) + //envp
		4 + // null
		4 + // auxv
		4 // null

	total := dataStart

	for _, str := range args {
		total += len(str) + 1
	}

	for _, str := range env {
		total += len(str) + 1
	}

	mem, err := vmem.Project(base, int32(total))
	if err != nil {
		return err
	}

	le := binary.LittleEndian

	le.PutUint32(mem, uint32(len(args)))

	nextStr := int32(dataStart)

	ptr := mem[4:]
	for _, str := range args {
		le.PutUint32(ptr, uint32(base+nextStr))
		copy(mem[nextStr:], []byte(str))
		mem[nextStr+int32(len(str))] = 0
		nextStr += int32(len(str) + 1)
		ptr = ptr[4:]
	}
	le.PutUint32(ptr, 0) // null after argv
	ptr = ptr[4:]

	for _, str := range env {
		le.PutUint32(ptr, uint32(base+nextStr))
		copy(mem[nextStr:], []byte(str))
		mem[nextStr+int32(len(str))] = 0
		nextStr += int32(len(str) + 1)
		ptr = ptr[4:]
	}

	le.PutUint32(ptr, 0) // null after envp
	ptr = ptr[4:]
	le.PutUint32(ptr, 0) // auxv
	ptr = ptr[4:]
	le.PutUint32(ptr, 0) // null after auxv

	return nil
}
