package columbia

import (
	"context"
	"encoding/binary"
	"io"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/evanphx/columbia/abi/linux"
	"github.com/evanphx/columbia/fs"
	hclog "github.com/hashicorp/go-hclog"
)

func sysWrite(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		fd  = args.Args.R0
		ptr = args.Args.R1
		sz  = args.Args.R2
	)

	var w io.Writer

	switch fd {
	case 1:
		w = os.Stdout
	case 2:
		w = os.Stderr
	default:
		return -1
	}

	data := make([]byte, sz)

	_, err := p.ReadAt(data, int64(ptr))
	if err != nil {
		return -1
	}

	n, err := w.Write(data)
	if err != nil {
		return -1
	}

	return int32(n)
}

func sysWritev(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		fd  = args.Args.R0
		iov = args.Args.R1
		cnt = args.Args.R2
	)

	var w io.Writer

	switch fd {
	case 1:
		w = os.Stdout
	case 2:
		w = os.Stderr
	default:
		return -1
	}

	tmp := make([]byte, 8)

	var ret int32

	for i := int32(0); i < cnt; i++ {
		_, err := p.ReadAt(tmp, int64(iov+(i*8)))
		if err != nil {
			return -1
		}

		ptr := binary.LittleEndian.Uint32(tmp)
		sz := binary.LittleEndian.Uint32(tmp[4:])

		// l.Info("read iov", "ptr", ptr, "sz", sz)

		data := make([]byte, sz)

		x, err := p.ReadAt(data, int64(ptr))
		if err != nil {
			return -1
		}

		ret += int32(x)

		w.Write(data)
	}

	return ret
}

func sysRead(l hclog.Logger, p *Process, args sysArgs) int32 {
	// var (
	// fd  = args.Args.R0
	// buf = args.Args.R1
	// sz  = args.Args.R2
	// )

	return -1
}

func sysStat64(l hclog.Logger, p *Process, args sysArgs) int32 {
	var (
		ptr = args.Args.R0
		buf = args.Args.R1
	)

	path, err := p.ReadCString(ptr)
	if err != nil {
		l.Error("error reading stat path", "error", err)
		return -ENOSYS
	}

	l.Info("stat64", "path", string(path))

	ctx := context.Background() // TODO pass ctx into the handles

	dentry, err := p.mount.LookupPath(ctx, string(path))
	if err != nil {
		if err == fs.ErrUnknownPath {
			return -ENOENT
		}

		l.Error("error looking up path", "error", err)
		return -ENOSYS
	}

	i := dentry.Inode

	us, err := i.Ops.UnstableAttr(ctx, i)
	if err != nil {
		l.Error("unable to retrieve unstable inode attrs", "error", err)
		return -ENOSYS
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

	spew.Dump(sb)

	err = p.CopyOut(buf, sb)
	if err != nil {
		l.Error("error copying out stat struct", "error", err)
		return -EINVAL
	}

	return 0
}

func init() {
	syscalls[4] = sysWrite
	syscalls[146] = sysWritev

	syscalls[3] = sysRead

	syscalls[195] = sysStat64
}
