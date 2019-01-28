package columbia

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/evanphx/columbia/exec"
	"github.com/evanphx/columbia/fs"
	"github.com/evanphx/columbia/fs/tarfs"
)

type Process struct {
	*exec.Process

	mount *fs.MountNamespace
	mem   *VirtualMemory
}

func (p *Process) SetupTar(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer f.Close()

	tf, err := tarfs.NewTarFS(f)
	if err != nil {
		return err
	}

	root, err := tf.Root()
	if err != nil {
		return err
	}

	p.mount = fs.NewMountNamespace()
	p.mount.Root = &fs.Dirent{
		Inode: root,
	}

	return nil
}

func (p *Process) ReadCString(ptr int32) ([]byte, error) {
	var buf bytes.Buffer

	var t [1]byte

	off := int64(ptr)

	for {
		_, err := p.ReadAt(t[:], off)
		if err != nil {
			return nil, err
		}

		if t[0] == 0 {
			break
		}

		buf.WriteByte(t[0])
		off += 1
	}

	return buf.Bytes(), nil
}

func (p *Process) CopyOut(addr int32, val interface{}) error {
	return binary.Write(writeAdapter{sub: p, offset: int64(addr)}, binary.LittleEndian, val)
}
