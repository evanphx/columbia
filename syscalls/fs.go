package syscalls

import (
	"context"

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
	var (
		ptr = args.Args.R0
		buf = args.Args.R1
	)

	path, err := p.ReadCString(ptr)
	if err != nil {
		l.Error("error reading stat path", "error", err)
		return -kernel.ENOSYS
	}

	dentry, err := p.Mount.LookupPath(ctx, string(path))
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
		Blocks:  int32(us.Size / i.StableAttr.BlockSize),
		ATime:   linux.NsecToTimespec(us.AccessTime.UnixNano()),
		MTime:   linux.NsecToTimespec(us.ModificationTime.UnixNano()),
		CTime:   linux.NsecToTimespec(us.StatusChangeTime.UnixNano()),
	}

	err = p.CopyOut(buf, sb)
	if err != nil {
		l.Error("error copying out stat struct", "error", err)
		return -kernel.EINVAL
	}

	return 0
}

func init() {
	Syscalls[5] = sysOpen
	Syscalls[195] = sysStat64

}
