package cancellable

import (
	"fmt"
	"io"

	"golang.org/x/net/context"
)

type CancelledErr struct {
	Orig error
}

func (e CancelledErr) Error() string {
	return fmt.Sprintf("Context was cancelled during IO: %v", e.Orig)
}

func IsCancelledErr(e error) bool {
	if e == nil {
		return false
	}

	_, ok := e.(CancelledErr)
	return ok
}

func Read(ctx context.Context, r io.Reader, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, CancelledErr{err}
	}

	var n int
	var err error
	complete := make(chan struct{})
	go func() {
		n, err = r.Read(p)
		close(complete)
	}()

	select {
	case <-complete:
		return n, err

	case <-ctx.Done():
		return 0, CancelledErr{ctx.Err()}
	}
}
