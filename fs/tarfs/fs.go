package tarfs

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/evanphx/columbia/abi/linux"
	"github.com/evanphx/columbia/device"
	"github.com/evanphx/columbia/fs"
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

type Dir struct {
	fs.StandardDirOps
	Unstable fs.InodeUnstableAttr
	Children map[string]*fs.Inode
	Order    []string
}

func (d *Dir) AddChild(name string, inode *fs.Inode) {
	d.Children[name] = inode
	d.Order = append(d.Order, name)
}

type File struct {
	fs.StandardFileOps
	Unstable fs.InodeUnstableAttr
	Body     []byte
}

type TarFS struct {
	Device *device.Device
	root   *fs.Inode
}

func findParent(root *Dir, name string) (*Dir, error) {
	dirName := filepath.Dir(name)

	if dirName == "" || dirName == "." {
		return root, nil
	}

	parts := strings.Split(dirName, "/")

	parent := root

	for _, sec := range parts {
		ch, ok := parent.Children[sec]
		if !ok {
			ch := &Dir{
				Children: make(map[string]*fs.Inode),
			}
			parent.Children[sec] = fs.NewInode(ch)
		}

		dir, ok := ch.Ops.(*Dir)
		if !ok {
			return nil, fs.ErrNotDirectory
		}

		parent = dir
	}

	return parent, nil
}

func NewTarFS(r io.Reader) (*TarFS, error) {
	tr := tar.NewReader(r)

	dev := device.NewAnonDevice()

	t := &TarFS{Device: dev}

	root := &Dir{
		Children: make(map[string]*fs.Inode),
	}

	var rootInode *fs.Inode

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
		us.AccessTime = linux.TimeToTimespec(hdr.AccessTime)
		us.ModificationTime = linux.TimeToTimespec(hdr.ModTime)
		us.StatusChangeTime = linux.TimeToTimespec(hdr.ChangeTime)
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

		// root!
		if name == "./" || name == "." {
			rootInode = fs.NewInode(root)
			rootInode.StableAttr = attr
			root.Unstable = us
			continue
		}

		var ops fs.InodeOps

		if attr.Type == fs.Directory {
			name = name[:len(name)-1]
			ops = &Dir{
				Unstable: us,
				Children: make(map[string]*fs.Inode),
			}
		} else {
			if attr.Type == fs.Symlink {
				us.Size = int64(len(hdr.Linkname))
				data = []byte(hdr.Linkname)
			}

			ops = &File{
				Unstable: us,
				Body:     data,
			}
		}

		parent, err := findParent(root, name)
		if err != nil {
			return nil, err
		}

		inode := &fs.Inode{
			StableAttr: attr,
			Ops:        ops,
		}

		parent.AddChild(filepath.Base(name), inode)
	}

	if rootInode == nil {
		rootInode = fs.NewInode(root)
	}

	t.root = rootInode

	return t, nil
}

func (t *TarFS) Root() (*fs.Inode, error) {
	return t.root, nil
}

func (d *Dir) LookupChild(ctx context.Context, inode *fs.Inode, name string) (*fs.Inode, error) {
	inode, ok := d.Children[name]
	if !ok {
		return nil, fs.ErrUnknownPath
	}

	return inode, nil
}

func (d *Dir) UnstableAttr(ctx context.Context, inode *fs.Inode) (*fs.InodeUnstableAttr, error) {
	return &d.Unstable, nil
}

func (f *File) UnstableAttr(ctx context.Context, inode *fs.Inode) (*fs.InodeUnstableAttr, error) {
	return &f.Unstable, nil
}

func (f *File) ReadLink(ctx context.Context, inode *fs.Inode) (string, error) {
	if inode.StableAttr.Type != fs.Symlink {
		return "", fs.ErrNotSymlink
	}

	return string(f.Body), nil
}

func (f *File) Reader(inode *fs.Inode) (io.ReadSeeker, error) {
	return bytes.NewReader(f.Body), nil
}

func (d *Dir) ReadDir(ctx context.Context, inode *fs.Inode, offset int, emit fs.ReadDirEmit) error {
	children := d.Order[offset:]
	if len(children) == 0 {
		return nil
	}

	for _, ent := range children {
		if !emit.EmitEntry(ent, d.Children[ent]) {
			break
		}
	}

	return nil
}
