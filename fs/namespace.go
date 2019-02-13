package fs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
)

type MountNamespace struct {
	Root        *Dirent
	DirentCache *lru.ARCCache
}

func NewMountNamespace() *MountNamespace {
	cache, err := lru.NewARC(1000) // TODO a better value
	if err != nil {
		panic(err)
	}

	return &MountNamespace{
		DirentCache: cache,
	}
}

func (m *MountNamespace) SetRoot(i *Inode) {
	m.Root = &Dirent{Inode: i}
}

func (m *MountNamespace) LookupPath(ctx context.Context, path string) (*Dirent, error) {
	dirent, err := m.LookupDirent(ctx, path)
	if err != nil {
		return nil, err
	}

	if dirent.Inode.StableAttr.Type != Symlink {
		return dirent, nil
	}

	target, err := dirent.Inode.Ops.ReadLink(ctx, dirent.Inode)
	if err != nil {
		return nil, err
	}

	fullTarget := filepath.Clean(filepath.Join(filepath.Dir(path), target))

	return m.LookupPath(ctx, fullTarget)
}

func (m *MountNamespace) LookupDirent(ctx context.Context, path string) (*Dirent, error) {
	if path[0] == '/' {
		path = path[1:]
	}

	if path == "" {
		return m.Root, nil
	}

	if val, ok := m.DirentCache.Get(path); ok {
		return val.(*Dirent), nil
	}

	sections := strings.Split(path, "/")

	cur := m.Root

	for _, part := range sections {
		if cur.Inode.StableAttr.Type != Directory {
			return nil, errors.Wrapf(ErrNotDirectory, "component: %s", cur.Name)
		}

		i, err := cur.Inode.Ops.LookupChild(ctx, cur.Inode, part)
		if err != nil {
			return nil, err
		}

		cur = &Dirent{Inode: i, Parent: cur, Name: part}
	}

	m.DirentCache.Add(path, cur)

	return cur, nil
}
