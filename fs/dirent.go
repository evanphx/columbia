package fs

import "io"

type Dirent struct {
	Name   string
	Parent *Dirent
	Inode  *Inode
}

func (d *Dirent) Reader() (io.ReadSeeker, error) {
	return d.Inode.Ops.Reader(d.Inode)
}
