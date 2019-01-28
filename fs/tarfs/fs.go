package tarfs

import (
	"archive/tar"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/evanphx/columbia/device"
	"github.com/evanphx/columbia/fs"
	hclog "github.com/hashicorp/go-hclog"
)

type entry struct {
	hdr          *tar.Header
	inode        *fs.Inode
	unstableAttr fs.InodeUnstableAttr
	body         []byte
}

func (e *entry) String() string {
	return spew.Sdump(e.inode)
}

type TarFS struct {
	Device  *device.Device
	entries map[string]*entry
}

func NewTarFS(r io.Reader) (*TarFS, error) {
	tr := tar.NewReader(r)

	dev := device.NewAnonDevice()
	entries := make(map[string]*entry)

	t := &TarFS{Device: dev}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, err
		}

		var attr fs.InodeStableAttr
		attr.BlockSize = 4096
		attr.DeviceFileMajor = uint16(dev.Major)
		attr.DeviceFileMinor = uint32(dev.Minor)
		attr.DeviceID = dev.DeviceID()
		attr.InodeID = dev.NextIno()
		attr.SetType(hdr.FileInfo().Mode())

		var us fs.InodeUnstableAttr
		us.AccessTime = hdr.AccessTime
		us.ModificationTime = hdr.ModTime
		us.StatusChangeTime = hdr.ChangeTime
		us.GroupId = hdr.Gid
		us.UserId = hdr.Uid
		us.Perms = int(os.FileMode(hdr.Mode).Perm())
		us.Size = hdr.Size

		name := hdr.Name

		if len(name) > 2 && name[:2] == "./" {
			name = name[2:]
		}

		if len(name) >= 1 && name[0] == '/' {
			name = name[1:]
		}

		if attr.Type == fs.Directory {
			name = name[:len(name)-1]
		}

		inode := &fs.Inode{
			StableAttr:    attr,
			MountRelative: name,
			Ops:           t,
		}

		entries[name] = &entry{hdr, inode, us, data}
	}

	t.entries = entries

	return t, nil
}

func (t *TarFS) Root() (*fs.Inode, error) {
	return &fs.Inode{
		StableAttr: fs.InodeStableAttr{
			Type: fs.Directory,
		},
		Ops: t,
	}, nil
}

func (t *TarFS) LookupChild(ctx context.Context, inode *fs.Inode, name string) (*fs.Inode, error) {
	path := filepath.Join(inode.MountRelative, name)

	entry, ok := t.entries[path]
	hclog.L().Info("tar lookup", "path", path, "found", ok)

	if !ok {
		return nil, fs.ErrUnknownPath
	}

	return entry.inode, nil
}

func (t *TarFS) UnstableAttr(ctx context.Context, inode *fs.Inode) (*fs.InodeUnstableAttr, error) {
	entry, ok := t.entries[inode.MountRelative]
	if !ok {
		return nil, fs.ErrUnknownPath
	}

	return &entry.unstableAttr, nil
}

func (t *TarFS) ReadLink(ctx context.Context, inode *fs.Inode) (string, error) {
	if inode.StableAttr.Type != fs.Symlink {
		return "", fs.ErrNotSymlink
	}

	entry, ok := t.entries[inode.MountRelative]
	if !ok {
		return "", fs.ErrUnknownPath
	}

	return entry.hdr.Linkname, nil
}
