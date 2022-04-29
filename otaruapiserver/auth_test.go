package otaruapiserver_test

// auth_test.go is conceptually the tests for github.com/nyaxt/otaru/apiserve/jwt
// , but since it relies on otaruapiserver.SystemService, it is placed here.

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	ojwt "github.com/nyaxt/otaru/apiserver/jwt"
	jwt_testutils "github.com/nyaxt/otaru/apiserver/jwt/jwt_testutils"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/otaruapiserver"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/testutils/testca"
)

const testListenAddr = "localhost:30246"

func init() {
	testutils.EnsureLogger()
}

type testServer struct {
	fs         *filesystem.FileSystem
	inodeRead  inodedb.ID
	inodeWrite inodedb.ID
	closeC     chan struct{}
	joinC      chan struct{}
}

func runTestServer(t *testing.T, pubkey *ecdsa.PublicKey) *testServer {
	t.Helper()

	ts := &testServer{
		fs:     testutils.TestFileSystem(),
		closeC: make(chan struct{}),
		joinC:  make(chan struct{}),
	}

	if err := ts.fs.WriteFile("/foo.txt", testutils.HelloWorld, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	inodeRead, err := ts.fs.FindNodeFullPath("/foo.txt")
	if err != nil {
		t.Fatalf("FindNodeFullPath: %v", err)
	}
	inodeWrite, err := ts.fs.CreateFileFullPath("/hoge.txt", 0644, 1000, 1000, time.Unix(1545811015, 0))
	if err != nil {
		t.Fatalf("CreateFileFullPath: %v", err)
	}
	ts.inodeRead = inodeRead
	ts.inodeWrite = inodeWrite

	jwtauth := ojwt.NoJWTAuth
	if pubkey != nil {
		jwtauth = ojwt.NewJWTAuthProvider(pubkey)
	}
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testListenAddr),
			apiserver.X509KeyPair(testca.CertPEM, testca.KeyPEM),
			apiserver.CloseChannel(ts.closeC),
			apiserver.JWTAuthProvider(jwtauth),
			otaruapiserver.InstallSystemService(),
			otaruapiserver.InstallFileHandler(ts.fs, jwtauth),
		); err != nil {
			panic(err)
		}
		close(ts.joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	return ts
}

func (ts *testServer) Terminate() {
	close(ts.closeC)
	<-ts.joinC
}

func testCliConfig(host *cli.Host) *cli.CliConfig {
	return &cli.CliConfig{Host: map[string]*cli.Host{
		"default": host,
	}}
}

func genHttpReadTest(cfg *cli.CliConfig, inode inodedb.ID, expectSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Fatalf("QueryConnectionInfo: %v", err)
		}

		r, err := cli.NewReaderHttpForTesting(ci, uint64(inode))
		if expectSuccess {
			if err != nil {
				t.Fatalf("cli.NewReaderHttpForTesting: %v", err)
			}
		} else {
			if err == nil {
				r.Close()
				t.Errorf("cli.NewReaderHttpForTesting unexpected success.")
				return
			}
			return
		}

		bs, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatalf("ioutil.ReadAll: %v", err)
		}

		if !bytes.Equal(bs, testutils.HelloWorld) {
			t.Errorf("unexpected bs: %v", bs)
		}

		r.Close()
	}
}

func genHttpWriteTest(cfg *cli.CliConfig, inode inodedb.ID, expectSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Errorf("QueryConnectionInfo: %v", err)
			return
		}

		w, err := cli.NewWriterHttpForTesting(ci, uint64(inode))
		if err != nil {
			t.Errorf("cli.NewWriterHttpForTesting: %v", err)
			return
		}

		_, err = w.Write(testutils.HogeFugaPiyo)
		if expectSuccess {
			if err != nil {
				t.Fatalf("w.Write: %v", err)
			}
		} else {
			if err != nil {
				w.Close()
				return
			}
		}

		err = w.Close()
		if expectSuccess {
			if err != nil {
				t.Errorf("w.Close: %v", err)
			}
		} else {
			if err == nil {
				t.Errorf("cli.NewReaderHttpForTesting unexpected success.")
				return
			}
		}
	}
}

func TestAuth_NoAuth(t *testing.T) {
	ts := runTestServer(t, nil)
	defer ts.Terminate()

	cfg := testCliConfig(&cli.Host{
		ApiEndpoint: testListenAddr,
		CACert:      testca.CACert,
	})

	t.Run("grpc", func(t *testing.T) {
		ctx := context.Background()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Fatalf("QueryConnectionInfo: %v", err)
		}

		conn, err := ci.DialGrpc(ctx)
		if err != nil {
			t.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestAdmin: %v.", err)
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestReadOnly: %v.", err)
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestAnonymous: %v.", err)
		}
	})

	t.Run("httpRead", genHttpReadTest(cfg, ts.inodeRead, true))
	t.Run("httpWrite", genHttpWriteTest(cfg, ts.inodeWrite, true))

}

func TestAuth_NoToken(t *testing.T) {
	ts := runTestServer(t, jwt_testutils.Pubkey)
	defer ts.Terminate()

	cfg := testCliConfig(&cli.Host{
		ApiEndpoint: testListenAddr,
		CACert:      testca.CACert,
	})

	t.Run("grpc", func(t *testing.T) {
		ctx := context.Background()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Fatalf("QueryConnectionInfo: %v", err)
		}

		conn, err := ci.DialGrpc(ctx)
		if err != nil {
			t.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestReadOnly should fail.")
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestAnonymous: %v.", err)
		}
	})

	t.Run("httpRead", genHttpReadTest(cfg, ts.inodeRead, false))
	t.Run("httpWrite", genHttpWriteTest(cfg, ts.inodeWrite, false))
}

func TestAuth_ValidReadOnlyToken(t *testing.T) {
	ts := runTestServer(t, jwt_testutils.Pubkey)
	defer ts.Terminate()

	cfg := testCliConfig(&cli.Host{
		ApiEndpoint: testListenAddr,
		CACert:      testca.CACert,
		AuthToken:   jwt_testutils.ReadOnlyToken,
	})

	t.Run("grpc", func(t *testing.T) {
		ctx := context.Background()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Fatalf("QueryConnectionInfo: %v", err)
		}

		conn, err := ci.DialGrpc(ctx)
		if err != nil {
			t.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestReadOnly: %v.", err)
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
			t.Errorf("AuthTestAnonymous: %v.", err)
		}
	})

	t.Run("httpRead", genHttpReadTest(cfg, ts.inodeRead, true))
	t.Run("httpWrite", genHttpWriteTest(cfg, ts.inodeWrite, false))
}

func TestAuth_AlgNoneToken(t *testing.T) {
	ts := runTestServer(t, jwt_testutils.Pubkey)
	defer ts.Terminate()

	cfg := testCliConfig(&cli.Host{
		ApiEndpoint: testListenAddr,
		CACert:      testca.CACert,
		AuthToken:   jwt_testutils.AlgNoneToken,
	})
	t.Run("grpc", func(t *testing.T) {
		ctx := context.Background()

		ci, err := cli.QueryConnectionInfo(cfg, "default")
		if err != nil {
			t.Fatalf("QueryConnectionInfo: %v", err)
		}

		conn, err := ci.DialGrpc(ctx)
		if err != nil {
			t.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestReadOnly should fail.")
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err == nil {
			t.Errorf("AuthTestAnonymous should fail.")
		}
	})

	t.Run("httpRead", genHttpReadTest(cfg, ts.inodeRead, false))
	t.Run("httpWrite", genHttpWriteTest(cfg, ts.inodeWrite, false))
}
