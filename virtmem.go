package columbia

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
)

const wasmPageSize = 65536 // (64 KB)

type Region struct {
	Start, Size int32

	linear []byte
}

func (reg *Region) Contains(x int32) bool {
	if x < reg.Start {
		return false
	}

	if x >= reg.Start+reg.Size {
		return false
	}

	return true
}

func pageRound(sz int32) int32 {
	if sz < wasmPageSize {
		return wasmPageSize
	}

	diff := sz % wasmPageSize
	if diff == 0 {
		return sz
	}

	return sz + (wasmPageSize - diff)
}

func (reg *Region) Project(addr, sz int32) []byte {
	offset := addr - reg.Start

	if len(reg.linear) == 0 {
		reg.linear = make([]byte, pageRound(offset+sz))
	}

	if len(reg.linear) < int(offset+sz) {
		slice := make([]byte, pageRound(offset+sz))
		copy(slice, reg.linear)

		reg.linear = slice
	}

	return reg.linear[offset : offset+sz]
}

type VirtualMemory struct {
	regions []*Region

	nextMmapStart int32
	size          int32
}

func NewVirtualMemory() *VirtualMemory {
	return &VirtualMemory{
		nextMmapStart: 0x10000,
	}
}

func (vm *VirtualMemory) Size() int {
	return int(vm.size)
}

func (vm *VirtualMemory) FindRegion(addr int32) (*Region, bool) {
	for _, reg := range vm.regions {
		if reg.Contains(addr) {
			return reg, true
		}
	}

	return nil, false
}

var ErrInvalidMemoryAccess = errors.New("invalid memory access via projection")

func (vm *VirtualMemory) Project(addr, sz int32) ([]byte, error) {
	reg, ok := vm.FindRegion(addr)
	if !ok {
		return nil, errors.Wrapf(ErrInvalidMemoryAccess, "error projecting address=%x, size=%x", addr, sz)
	}

	return reg.Project(addr, sz), nil
}

func (vm *VirtualMemory) Grow(additional int32) error {
	reg, ok := vm.FindRegion(0)
	if !ok {
		return ErrInvalidMemoryAccess
	}

	reg.Size += additional

	return nil
}

var ErrBadRegionRequest = errors.New("bad region request")

func (vm *VirtualMemory) NewRegion(addr, size int32) (*Region, error) {
	if addr == -1 {
		addr = vm.nextMmapStart
		vm.nextMmapStart += pageRound(size + (1024 * 1024)) // TODO: something better?
	} else {
		reg, ok := vm.FindRegion(addr)
		if ok {
			if reg.Size < size {
				return nil, ErrBadRegionRequest
			}

			return reg, nil
		}
	}

	hclog.L().Info("virtmem: new region", "addr", addr, "size", size)

	reg := &Region{
		Start: addr,
		Size:  size,
	}

	vm.regions = append(vm.regions, reg)

	vm.size += size

	if reg.Contains(vm.nextMmapStart) {
		vm.nextMmapStart = pageRound(addr + size)
	}

	return reg, nil
}
