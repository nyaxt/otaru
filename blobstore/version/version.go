package version

import "io"

// FIXME: handle overflows
type Version int64
type QueryFunc func(r io.Reader) (Version, error)
