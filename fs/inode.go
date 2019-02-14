package fs

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/evanphx/columbia/abi/linux"
)

var (
	ErrUnknownPath    = errors.New("unknown path")
	ErrNotSymlink     = errors.New("not symlink")
	ErrNotDirectory   = errors.New("not a directory")
	ErrNotImplemented = errors.New("not implemented")
)

// InodeType enumerates types of Inodes.
type InodeType int

const (
	// RegularFile is a regular file.
	RegularFile InodeType = iota

	// SpecialFile is a file that doesn't support SeekEnd. It is used for
	// things like proc files.
	SpecialFile

	// Directory is a directory.
	Directory

	// SpecialDirectory is a directory that *does* support SeekEnd. It's
	// the opposite of the SpecialFile scenario above. It similarly
	// supports proc files.
	SpecialDirectory

	// Symlink is a symbolic link.
	Symlink

	// Pipe is a pipe (named or regular).
	Pipe

	// Socket is a socket.
	Socket

	// CharacterDevice is a character device.
	CharacterDevice

	// BlockDevice is a block device.
	BlockDevice

	// Anonymous is an anonymous type when none of the above apply.
	// Epoll fds and event-driven fds fit this category.
	Anonymous
)

// String returns a human-readable representation of the InodeType.
func (n InodeType) String() string {
	switch n {
	case RegularFile, SpecialFile:
		return "file"
	case Directory, SpecialDirectory:
		return "directory"
	case Symlink:
		return "symlink"
	case Pipe:
		return "pipe"
	case Socket:
		return "socket"
	case CharacterDevice:
		return "character-device"
	case BlockDevice:
		return "block-device"
	case Anonymous:
		return "anonymous"
	default:
		return "unknown"
	}
}

type InodeStableAttr struct {
	// Type is the InodeType of a InodeOperations.
	Type InodeType

	// DeviceID is the device on which a InodeOperations resides.
	DeviceID uint64

	// InodeID uniquely identifies InodeOperations on its device.
	InodeID uint64

	// BlockSize is the block size of data backing this InodeOperations.
	BlockSize int64

	// DeviceFileMajor is the major device number of this Node, if it is a
	// device file.
	DeviceFileMajor uint16

	// DeviceFileMinor is the minor device number of this Node, if it is a
	// device file.
	DeviceFileMinor uint32
}

func (attr *InodeStableAttr) SetType(mode os.FileMode) {
	switch mode & os.ModeType {
	case 0:
		attr.Type = RegularFile
	case os.ModeDir:
		attr.Type = Directory
	case os.ModeSymlink:
		attr.Type = Symlink
	case os.ModeNamedPipe:
		attr.Type = Pipe
	case os.ModeSocket:
		attr.Type = Socket
	case os.ModeCharDevice:
		attr.Type = CharacterDevice
	case os.ModeDevice:
		attr.Type = BlockDevice
	default:
		attr.Type = Anonymous
	}
}

// UnstableAttr contains Inode attributes that may change over the lifetime
// of the Inode.
//
type InodeUnstableAttr struct {
	// Size is the file size in bytes.
	Size int64

	// Usage is the actual data usage in bytes.
	Usage int64

	// Perms is the protection (read/write/execute for user/group/other).
	Perms int

	UserId, GroupId int

	// AccessTime is the time of last access
	AccessTime linux.Timespec

	// ModificationTime is the time of last modification.
	ModificationTime linux.Timespec

	// StatusChangeTime is the time of last attribute modification.
	StatusChangeTime linux.Timespec

	// Links is the number of hard links.
	Links uint64
}

type InodeOps interface {
	LookupChild(ctx context.Context, inode *Inode, name string) (*Inode, error)
	UnstableAttr(ctx context.Context, inode *Inode) (*InodeUnstableAttr, error)
	ReadLink(ctx context.Context, inode *Inode) (string, error)
	Reader(inode *Inode) (io.ReadSeeker, error)
	ReadDir(ctx context.Context, inode *Inode, offset int, emit ReadDirEmit) error
}

type Inode struct {
	StableAttr InodeStableAttr
	Ops        InodeOps
}

func NewInode(attr InodeStableAttr, ops InodeOps) *Inode {
	return &Inode{
		StableAttr: attr,
		Ops:        ops,
	}
}
