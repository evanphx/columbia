package fs

import (
	"context"
	"io"
)

type StandardDirOps struct{}

func (_ StandardDirOps) ReadLink(ctx context.Context, inode *Inode) (string, error) {
	return "", ErrNotSymlink
}

func (_ StandardDirOps) Reader(inode *Inode) (io.ReadSeeker, error) {
	return nil, ErrNotImplemented
}

type StandardFileOps struct{}

func (_ StandardFileOps) LookupChild(ctx context.Context, inode *Inode, name string) (*Inode, error) {
	return nil, ErrNotImplemented
}

func (_ StandardFileOps) ReadDir(ctx context.Context, inode *Inode, offset int, emit ReadDirEmit) error {
	return ErrNotImplemented
}
