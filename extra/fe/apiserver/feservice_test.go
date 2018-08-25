package apiserver_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	opath "github.com/nyaxt/otaru/cli/path"
	feapiserver "github.com/nyaxt/otaru/extra/fe/apiserver"
	"github.com/nyaxt/otaru/extra/fe/pb"
	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/otaruapiserver"
	opb "github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/testutils"
)

type mockFsBackend struct {
	closeC chan struct{}
	joinC  chan struct{}
	fs     *filesystem.FileSystem
}

func runMockFsBackend() *mockFsBackend {
	fs := testutils.TestFileSystem()

	m := &mockFsBackend{
		closeC: make(chan struct{}),
		joinC:  make(chan struct{}),
		fs:     fs,
	}

	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testBeListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.CloseChannel(m.closeC),
			otaruapiserver.InstallFileSystemService(fs),
		); err != nil {
			panic(err)
		}
		close(m.joinC)
	}()

	return m
}

func (m *mockFsBackend) Terminate() {
	close(m.closeC)
	<-m.joinC
}

func TestFeService(t *testing.T) {
	rootPath, err := ioutil.TempDir("", "felocal")
	if err != nil {
		t.Fatalf("failed to create tmpdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(rootPath, "foo"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(rootPath, "foo", "a.txt"), []byte("hoge\n"), 0644); err != nil {
		t.Fatalf("a.txt: %v", err)
	}

	cfg := &cli.CliConfig{
		Host: map[string]*cli.Host{
			"hostfoo": &cli.Host{
				ApiEndpoint:      testBeListenAddr,
				ExpectedCertFile: certFile,
			},
		},
		LocalRootPath: rootPath,
	}

	m := runMockFsBackend()

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testFeListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.CloseChannel(closeC),
			feapiserver.InstallFeService(cfg),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	certtext, err := ioutil.ReadFile(certFile)
	if err != nil {
		t.Fatalf("Failed to read specified cert file: %s", certFile)
	}
	tc, err := cli.TLSConfigFromCertText(certtext)
	if err != nil {
		t.Fatalf("TLSConfigFromCertText: %v", err)
	}
	conn, err := cli.DialGrpc(testFeListenAddr, tc)
	if err != nil {
		t.Fatalf("DialGrpcVhost: %v", err)
	}
	fesc := pb.NewFeServiceClient(conn)

	t.Run("ListHosts", func(tt *testing.T) {
		ctx := context.TODO()
		resp, err := fesc.ListHosts(ctx, &pb.ListHostsRequest{})
		if err != nil {
			tt.Fatalf("ListHosts: %v", err)
		}
		if !reflect.DeepEqual(resp.Host, []string{"hostfoo", opath.VhostLocal}) {
			tt.Errorf("ListHosts resp: %v", resp.Host)
		}
	})
	t.Run("ListLocalDir", func(tt *testing.T) {
		ctx := context.TODO()
		resp, err := fesc.ListLocalDir(ctx, &pb.ListLocalDirRequest{
			Path: "/foo",
		})
		if err != nil {
			tt.Fatalf("ListLocalDir: %v", err)
		}
		if len(resp.Entry) != 1 {
			tt.Fatalf("resp.Entry: %+v", resp.Entry)
		}
		if resp.Entry[0].Name != "a.txt" {
			tt.Fatalf("resp.Entry: %+v", resp.Entry)
		}
	})
	t.Run("MkdirLocal", func(tt *testing.T) {
		if err := os.Mkdir(filepath.Join(rootPath, "mkdir"), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		ctx := context.TODO()
		_, err := fesc.MkdirLocal(ctx, &pb.MkdirLocalRequest{
			Path: "/mkdir/hoge",
		})
		if err != nil {
			tt.Fatalf("MkdirLocal: %v", err)
		}

		resp, err := fesc.ListLocalDir(ctx, &pb.ListLocalDirRequest{
			Path: "/mkdir",
		})
		if err != nil {
			tt.Fatalf("ListLocalDir: %v", err)
		}
		if len(resp.Entry) != 1 {
			tt.Fatalf("resp.Entry: %+v", resp.Entry)
		}
		if resp.Entry[0].Name != "hoge" {
			tt.Fatalf("resp.Entry: %+v", resp.Entry)
		}
		if resp.Entry[0].Type != opb.INodeType_DIR {
			tt.Fatalf("resp.Entry: %+v", resp.Entry)
		}
	})

	close(closeC)
	<-joinC

	m.Terminate()
}
