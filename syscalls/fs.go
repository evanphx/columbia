package syscalls

import (
	"context"
	"encoding/binary"
	"path/filepath"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/abi/linux"
	"github.com/evanphx/columbia/fs"
	"github.com/evanphx/columbia/kernel"
	hclog "github.com/hashicorp/go-hclog"
)

func sysOpen(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		ptr  = args.Args.R0
		mode = args.Args.R1
	)

	path, err := p.ReadCString(ptr)
	if err != nil {
		l.Error("error reading cstring", "error", err)
		return -1
	}

	l.Trace("open file", "path", string(path), "mode", mode)

	fd, err := p.OpenFile(ctx, string(path), int(mode))
	if err != nil {
		if err == fs.ErrUnknownPath {
			return -abi.ENOENT
		}

		l.Error("error opening file", "error", err)

		return -abi.ENOSYS
	}

	return int32(fd)
}

func sysStat64(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return stat(ctx, l, p, args, true)
}

func sysLstat64(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	return stat(ctx, l, p, args, false)
}

func stat(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs, resolve bool) int32 {
	var (
		ptr = args.Args.R0
		buf = args.Args.R1
	)

	path, err := p.ReadCString(ptr)
	if err != nil {
		l.Error("error reading stat path", "error", err)
		return -kernel.ENOSYS
	}

	abs := string(path)

	if !filepath.IsAbs(abs) {
		abs = filepath.Join(p.Curwd(), abs)
	}

	l.Trace("syscall/stat", "path", abs)

	var dentry *fs.Dirent

	if resolve {
		dentry, err = p.Mount.LookupPath(ctx, abs)
	} else {
		dentry, err = p.Mount.LookupDirent(ctx, abs)
	}
	if err != nil {
		if err == fs.ErrUnknownPath {
			return -kernel.ENOENT
		}

		l.Error("error looking up path", "error", err)
		return -kernel.ENOSYS
	}

	i := dentry.Inode

	us, err := i.Ops.UnstableAttr(ctx, i)
	if err != nil {
		l.Error("unable to retrieve unstable inode attrs", "error", err)
		return -kernel.ENOSYS
	}

	var mode uint32
	switch i.StableAttr.Type {
	case fs.RegularFile, fs.SpecialFile:
		mode |= linux.ModeRegular
	case fs.Symlink:
		mode |= linux.ModeSymlink
	case fs.Directory, fs.SpecialDirectory:
		mode |= linux.ModeDirectory
	case fs.Pipe:
		mode |= linux.ModeNamedPipe
	case fs.CharacterDevice:
		mode |= linux.ModeCharacterDevice
	case fs.BlockDevice:
		mode |= linux.ModeBlockDevice
	case fs.Socket:
		mode |= linux.ModeSocket
	}

	sb := linux.Stat{
		Dev:     uint64(linux.MakeDeviceID(i.StableAttr.DeviceFileMajor, i.StableAttr.DeviceFileMinor)),
		Ino:     i.StableAttr.InodeID,
		Mode:    mode | uint32(us.Perms),
		UID:     uint32(us.UserId),
		GID:     uint32(us.GroupId),
		Size:    us.Size,
		Blksize: i.StableAttr.BlockSize,
		Blocks:  int64(us.Size / 512),
		ATime:   us.AccessTime,
		MTime:   us.ModificationTime,
		CTime:   us.StatusChangeTime,
	}

	err = p.CopyOut(buf, sb)
	if err != nil {
		l.Error("error copying out stat struct", "error", err)
		return -kernel.EINVAL
	}

	return 0
}

type direntHeader struct {
	Inode  uint64
	Offset uint64
	Reclen uint16
	Type   uint8
}

type dirent struct {
	hdr  direntHeader
	name []byte
}

var direntHeaderSize = binary.Size(direntHeader{})

type direntEmitter struct {
	offset     uint64
	addr, left int32
	t          *kernel.Task
}

func (d *dirent) calculateReclen() uint16 {
	a := direntHeaderSize + len(d.name)
	r := (a + 4) &^ (4 - 1) // my head hurts.
	padding := r - a
	d.name = append(d.name, make([]byte, padding)...)
	d.hdr.Reclen = uint16(r)
	return uint16(r)
}

func (d *direntEmitter) EmitEntry(name string, inode *fs.Inode) bool {
	var typ uint8

	switch inode.StableAttr.Type {
	case fs.Symlink:
		typ = 10
	case fs.BlockDevice:
		typ = 6
	case fs.CharacterDevice:
		typ = 2
	case fs.Directory:
		typ = 4
	case fs.RegularFile:
		typ = 8
	}

	de := dirent{
		hdr: direntHeader{
			Inode:  inode.StableAttr.InodeID,
			Offset: d.offset,
			Type:   typ,
		},
		name: []byte(name),
	}

	if int32(de.calculateReclen()) > d.left {
		return false
	}

	err := d.t.CopyOut(d.addr, de.hdr)
	if err != nil {
		return false
	}

	d.addr += int32(direntHeaderSize)
	d.left -= int32(direntHeaderSize)

	err = d.t.CopyOut(d.addr, de.name)
	if err != nil {
		return false
	}

	d.addr += int32(len(de.name))
	d.left -= int32(len(de.name))

	d.offset++

	return true
}

func sysGetdents64(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		fd   = args.Args.R0
		ptr  = args.Args.R1
		size = args.Args.R2
	)

	file, ok := p.GetFile(int(fd))
	if !ok {
		return -abi.EINVAL
	}

	dc, ok := file.Context.(*kernel.DirContext)
	if !ok {
		return -abi.EINVAL
	}

	de := &direntEmitter{
		offset: uint64(dc.Offset),
		addr:   ptr,
		left:   size,
		t:      p,
	}

	err := file.Dirent.Inode.Ops.ReadDir(ctx, file.Dirent.Inode, dc.Offset, de)
	if err != nil {
		if err == context.Canceled {
			return -abi.EINTR
		}

		l.Error("error during readdir", "error", err)
		return -abi.ENOSYS
	}

	dc.Offset = int(de.offset)

	return size - de.left
}

func sysReadlink(ctx context.Context, l hclog.Logger, p *kernel.Task, args SysArgs) int32 {
	var (
		addr = args.Args.R0
		ptr  = args.Args.R1
		size = args.Args.R2
	)

	path, err := p.ReadCString(addr)
	if err != nil {
		return -abi.EFAULT
	}

	dirent, err := p.Mount.LookupDirent(ctx, string(path))
	if err != nil {
		if err == fs.ErrUnknownPath {
			return -abi.ENOENT
		}

		l.Error("error resolving dirent", "error", err)
		return -abi.ENOSYS
	}

	target, err := dirent.Inode.Ops.ReadLink(ctx, dirent.Inode)
	if err != nil {
		l.Error("error resolving link", "error", err)
		return -abi.ENOSYS
	}

	if len(target) > int(size) {
		target = target[:size]
	}

	err = p.CopyOut(ptr, []byte(target))
	if err != nil {
		l.Error("error copying data to userspace", "error", err)
		return -abi.EFAULT
	}

	return int32(len(target))
}

func init() {
	Syscalls[5] = sysOpen
	Syscalls[195] = sysStat64
	Syscalls[196] = sysLstat64
	Syscalls[220] = sysGetdents64
	Syscalls[85] = sysReadlink
}
