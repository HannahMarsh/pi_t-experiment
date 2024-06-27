package utils

import (
	"bytes"
	"compress/gzip"
	"io"
)

// compress compresses data using gzip
func Compress(data []byte) (bytes.Buffer, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return b, err
	}
	err = w.Close()
	if err != nil {
		return b, err
	}
	return b, nil
}

// decompress decompresses gzip data
func Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
