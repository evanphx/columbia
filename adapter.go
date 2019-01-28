package columbia

import "io"

type writeAdapter struct {
	sub    io.WriterAt
	offset int64
}

func (w writeAdapter) Write(b []byte) (int, error) {
	return w.sub.WriteAt(b, w.offset)
}
