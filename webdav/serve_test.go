package webdav

import (
	"testing"

	tu "github.com/nyaxt/otaru/testutils"
)

const testListenAddr = "localhost:20800"

func TestServe_Basic(t *testing.T) {
	apiCloseC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		err := Serve(
			FileSystem(tu.TestFileSystem()),
			ListenAddr(testListenAddr),
			CloseChannel(apiCloseC),
		)
		if err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		joinC <- struct{}{}
	}()
	close(apiCloseC)
	<-joinC
}
