package main

import (
	"io"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

type readWriterTo struct {
	api.Body
}

func (sr readWriterTo) Read(p []byte) (n int, err error) {
	size, eof := sr.Body.Read(p)
	if eof {
		err = io.EOF
	}
	n = int(size)
	return
}

// WriteTo implements io.WriterTo and it is handy for copying the body to the
// into the transaction buffer.
func (sr readWriterTo) WriteTo(w io.Writer) (n int64, err error) {
	size, err := sr.Body.WriteTo(w)
	return int64(size), err
}

var _ io.Reader = readWriterTo{}
var _ io.WriterTo = readWriterTo{}
