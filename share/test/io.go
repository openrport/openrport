package test

import (
	"io"

	"github.com/stretchr/testify/mock"
)

type ReadCloserMock struct {
	Reader io.Reader
	mock.Mock
}

func (rcm *ReadCloserMock) Read(p []byte) (n int, err error) {
	if rcm.Reader != nil {
		return rcm.Reader.Read(p)
	}

	args := rcm.Called(p)

	return args.Int(0), args.Error(1)
}

func (rcm *ReadCloserMock) Close() error {
	args := rcm.Called()

	return args.Error(0)
}

type ReadWriteCloserMock struct {
	ReadCloserMock
	Writer io.Writer
}

func (rcm *ReadWriteCloserMock) Write(p []byte) (n int, err error) {
	if rcm.Writer != nil {
		return rcm.Writer.Write(p)
	}

	args := rcm.Called(p)
	return args.Int(0), args.Error(1)
}
