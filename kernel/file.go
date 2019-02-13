package kernel

import (
	"io"
	"sync"

	"github.com/evanphx/columbia/fs"
)

type DirContext struct {
	Offset int
}

type File struct {
	mu          sync.Mutex
	refs        int
	CloseOnExec bool

	Dirent *fs.Dirent
	r      io.ReadCloser
	w      io.WriteCloser

	Context interface{}
}

func (f *File) Writer() (io.Writer, bool) {
	if f.w == nil {
		return nil, false
	}

	return f.w, true
}

func (f *File) Reader() (io.Reader, bool) {
	if f.r == nil {
		return nil, false
	}

	return f.r, true
}

func (f *File) incRef() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.refs++
}

func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.refs--
	if f.refs > 0 {
		return nil
	}

	var err error

	if f.r != nil {
		se := f.r.Close()
		if se != nil {
			err = se
		}
	}

	if f.w != nil {
		se := f.w.Close()
		if se != nil {
			err = se
		}
	}

	return err
}
