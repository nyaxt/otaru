package util

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func Gzip(p []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(p); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Gunzip(p []byte) ([]byte, error) {
	b := bytes.NewBuffer(p)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	u, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return u, nil
}
