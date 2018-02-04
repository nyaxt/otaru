package cli_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	tu "github.com/nyaxt/otaru/testutils"
)

const testListenAddr = "localhost:20247"

var cfg *cli.CliConfig
var testdir string

func TestMain(m *testing.M) {
	tu.EnsureLogger()

	otarudir := os.Getenv("OTARUDIR")
	certFile := path.Join(otarudir, "cert.pem")
	cfg = &cli.CliConfig{
		Host: map[string]*cli.Host{
			"default": &cli.Host{
				ApiEndpoint:      testListenAddr,
				ExpectedCertFile: certFile,
			},
		},
	}

	// populate test data
	var err error
	testdir, err = ioutil.TempDir("", "otaru-clitest")
	if err != nil {
		log.Panicf("tempdir: %v", err)
	}
	defer os.RemoveAll(testdir)

	if err := ioutil.WriteFile(filepath.Join(testdir, "hello.txt"), tu.HelloWorld, 0644); err != nil {
		log.Panicf("write hello.txt: %v", err)
	}

	os.Exit(m.Run())
}

func withApiServer(t *testing.T, f func()) {
	t.Helper()

	fs := tu.TestFileSystem()

	otarudir := os.Getenv("OTARUDIR")
	certFile := path.Join(otarudir, "cert.pem")
	keyFile := path.Join(otarudir, "cert-key.pem")

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.InstallFileSystemService(fs),
			apiserver.InstallFileHandler(fs),
			apiserver.CloseChannel(closeC),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	f()

	close(closeC)
	<-joinC
}

func TestBasic(t *testing.T) {
	withApiServer(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := cli.Put(ctx, cfg, []string{"put", filepath.Join(testdir, "hello.txt"), "/hello.txt"}); err != nil {
			t.Errorf("cli.Put failed: %v", err)
		}

		getpath := filepath.Join(testdir, "hello_get.txt")
		if err := cli.Get(ctx, cfg, []string{"get", "-o", getpath, "/hello.txt"}); err != nil {
			t.Errorf("cli.Get failed: %v", err)
		}
		b, err := ioutil.ReadFile(getpath)
		if err != nil {
			t.Errorf("ioutil.ReadAll(getpath): %v", err)
		}
		if !bytes.Equal(b, tu.HelloWorld) {
			t.Errorf("PRead content != PWrite content: %v", b)
		}

		var buf bytes.Buffer
		if err := cli.Ls(ctx, &buf, cfg, []string{"ls"}); err != nil {
			t.Errorf("cli.Ls failed: %v", err)
		}
		if string(buf.Bytes()) != "hello.txt\n" {
			t.Errorf("ls unexpectedly returned: %s", string(buf.Bytes()))
		}
	})
}

func TestPutVariations(t *testing.T) {
	withApiServer(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := cli.Put(ctx, cfg, []string{"put", filepath.Join(testdir, "hello.txt"), "/hello.txt"}); err != nil {
			t.Errorf("cli.Put failed: %v", err)
		}
		// FIXME: more tests
	})
}
