package host

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/evanphx/columbia/abi/linux"
	"github.com/evanphx/columbia/device"
	"github.com/evanphx/columbia/fs"
	"github.com/evanphx/columbia/log"
)

type HostFS struct {
	Device *device.Device
	root   *fs.Inode
}

func statToStableAttr(stat os.FileInfo) fs.InodeStableAttr {
	lower := stat.Sys().(*syscall.Stat_t)

	var attr fs.InodeStableAttr
	attr.BlockSize = int64(lower.Blksize)
	major, minor := linux.DecodeDeviceID(uint32(lower.Dev))
	attr.DeviceFileMajor = major
	attr.DeviceFileMinor = minor
	attr.DeviceID = uint64(lower.Dev)
	attr.InodeID = lower.Ino
	attr.SetType(stat.Mode())

	return attr
}

func NewHostFS(path string) (*HostFS, error) {
	dev := device.NewAnonDevice()
	h := &HostFS{
		Device: dev,
	}

	log.L.Trace("creating host fs", "path", path)

	stat, err := os.Lstat(path)
	if err != nil {
		log.L.Error("error stating hostfs path", "error", err)
		return nil, err
	}

	attr := statToStableAttr(stat)

	h.root = fs.NewInode(attr, &Dir{host: h, FSPath: FSPath{Path: path, Info: stat}})

	return h, nil
}

func (h *HostFS) Root() (*fs.Inode, error) {
	return h.root, nil
}

type FSPath struct {
	Path string
	Info os.FileInfo
}

func convertTS(ts syscall.Timespec) linux.Timespec {
	return linux.Timespec{
		Sec:  ts.Sec,
		Nsec: int32(ts.Nsec),
	}
}

func (p *FSPath) UnstableAttr(ctx context.Context, inode *fs.Inode) (*fs.InodeUnstableAttr, error) {
	stat, err := os.Lstat(p.Path)
	if err != nil {
		return nil, err
	}

	lower := stat.Sys().(*syscall.Stat_t)

	var us fs.InodeUnstableAttr
	us.AccessTime = convertTS(lower.Atimespec)
	us.ModificationTime = convertTS(lower.Mtimespec)
	us.StatusChangeTime = convertTS(lower.Ctimespec)
	us.GroupId = int(lower.Gid)
	us.UserId = int(lower.Uid)
	us.Perms = int(stat.Mode().Perm())
	us.Size = stat.Size()

	return &us, nil
}

type Dir struct {
	fs.StandardDirOps
	FSPath

	host *HostFS
}

type Entry struct {
	fs.StandardFileOps
	FSPath
}

func (e *Entry) ReadLink(ctx context.Context, inode *fs.Inode) (string, error) {
	return os.Readlink(e.Path)
}

func (e *Entry) Reader(inode *fs.Inode) (io.ReadSeeker, error) {
	return os.Open(e.Path)
}

func (d *Dir) LookupChild(ctx context.Context, inode *fs.Inode, name string) (*fs.Inode, error) {
	log.L.Trace("lookup child on host fs", "dir", d.Path, "name", name)

	cp := filepath.Join(d.Path, name)

	stat, err := os.Lstat(cp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fs.ErrUnknownPath
		}

		return nil, err
	}

	attr := statToStableAttr(stat)

	if stat.IsDir() {
		return fs.NewInode(attr, &Dir{host: d.host, FSPath: FSPath{Path: cp, Info: stat}}), nil
	}

	return fs.NewInode(attr, &Entry{FSPath: FSPath{Path: cp, Info: stat}}), nil
}

func (d *Dir) ReadDir(ctx context.Context, inode *fs.Inode, offset int, emit fs.ReadDirEmit) error {
	infos, err := ioutil.ReadDir(d.Path)
	if err != nil {
		return err
	}

	infos = infos[offset:]
	if len(infos) == 0 {
		return nil
	}

	for _, ent := range infos {
		attr := statToStableAttr(ent)
		inode := fs.NewInode(attr, &Entry{FSPath: FSPath{Path: filepath.Join(d.Path, ent.Name()), Info: ent}})

		if !emit.EmitEntry(ent.Name(), inode) {
			break
		}
	}

	return nil
}
