package main

import "io"

type stdReader struct {
	reader
}

func (sr *stdReader) Read(p []byte) (n int, err error) {
	size, eof := sr.reader.Read(p)
	if eof {
		err = io.EOF
	}
	n = int(size)
	return
}

var _ io.Reader = &stdReader{}

type reader interface {
	Read([]byte) (size uint32, eof bool)
}
