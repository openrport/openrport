package test

import "io"

type ReadWriter struct {
	io.Reader
	io.WriteCloser
}

func NewMockChannel() (*ReadWriter, *ReadWriter) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	return &ReadWriter{
			Reader:      r1,
			WriteCloser: w2,
		}, &ReadWriter{
			Reader:      r2,
			WriteCloser: w1,
		}
}
