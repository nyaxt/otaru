package otaruapiserver_test

// auth_test.go is conceptually the tests for github.com/nyaxt/otaru/apiserve/jwt
// , but since it relies on otaruapiserver.SystemService, it is placed here.

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path"
	"testing"
	"time"

	jwt "gopkg.in/dgrijalva/jwt-go.v3"

	"github.com/nyaxt/otaru/apiserver"
	ojwt "github.com/nyaxt/otaru/apiserver/jwt"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/otaruapiserver"
	"github.com/nyaxt/otaru/pb"
	"github.com/nyaxt/otaru/testutils"
)

const testListenAddr = "localhost:30246"

var certFile string
var keyFile string

var testKey *ecdsa.PrivateKey
var testPubkey *ecdsa.PublicKey
var testReadOnlyToken string
var testAlgNoneToken string

func init() {
	testutils.EnsureLogger()

	otarudir := os.Getenv("OTARUDIR")
	certFile = path.Join(otarudir, "cert.pem")
	keyFile = path.Join(otarudir, "cert-key.pem")

	var err error
	testKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("ecdsa.GenerateKey")
	}
	testPubkey = &testKey.PublicKey

	now := time.Now()

	claims := ojwt.Claims{
		Role: ojwt.RoleReadOnly.String(),
		StandardClaims: jwt.StandardClaims{
			Audience:  ojwt.OtaruAudience,
			ExpiresAt: (now.Add(time.Hour)).Unix(),
			Issuer:    "auth_test",
			NotBefore: now.Unix(),
			Subject:   "auth_test",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	testReadOnlyToken, err = token.SignedString(testKey)
	if err != nil {
		panic("testReadOnlyToken")
	}

	token = jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	testAlgNoneToken, err = token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		panic("testAlgNoneToken")
	}
}

type testServer struct {
	closeC chan struct{}
	joinC  chan struct{}
}

func runTestServer(pubkey *ecdsa.PublicKey) *testServer {
	ts := &testServer{
		closeC: make(chan struct{}),
		joinC:  make(chan struct{}),
	}

	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.CloseChannel(ts.closeC),
			apiserver.JWTPublicKey(pubkey),
			otaruapiserver.InstallSystemService(),
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

func TestAuth_NoAuth(t *testing.T) {
	ts := runTestServer(nil)

	ci, err := cli.ConnectionInfoFromHost(&cli.Host{
		ApiEndpoint:      testListenAddr,
		ExpectedCertFile: certFile,
	})
	if err != nil {
		t.Fatalf("ConnectionInfoFromHost: %v", err)
	}

	conn, err := ci.DialGrpc()
	if err != nil {
		t.Fatalf("DialGrpc: %v", err)
	}
	defer conn.Close()

	ssc := pb.NewSystemInfoServiceClient(conn)

	ctx := context.Background()

	if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err != nil {
		t.Errorf("AuthTestAdmin: %v.", err)
	}

	if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err != nil {
		t.Errorf("AuthTestReadOnly: %v.", err)
	}

	if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
		t.Errorf("AuthTestAnonymous: %v.", err)
	}

	ts.Terminate()
}

func TestAuth_Enabled(t *testing.T) {
	ts := runTestServer(testPubkey)

	t.Run("NoToken", func(tt *testing.T) {
		ci, err := cli.ConnectionInfoFromHost(&cli.Host{
			ApiEndpoint:      testListenAddr,
			ExpectedCertFile: certFile,
		})
		if err != nil {
			tt.Fatalf("ConnectionInfoFromHost: %v", err)
		}

		conn, err := ci.DialGrpc()
		if err != nil {
			tt.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		ctx := context.Background()

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestReadOnly should fail.")
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
			tt.Errorf("AuthTestAnonymous: %v.", err)
		}
	})

	t.Run("ValidToken", func(tt *testing.T) {
		ci, err := cli.ConnectionInfoFromHost(&cli.Host{
			ApiEndpoint:      testListenAddr,
			ExpectedCertFile: certFile,
			AuthToken:        testReadOnlyToken,
		})
		if err != nil {
			tt.Fatalf("ConnectionInfoFromHost: %v", err)
		}

		conn, err := ci.DialGrpc()
		if err != nil {
			tt.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		ctx := context.Background()

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err != nil {
			tt.Errorf("AuthTestReadOnly: %v.", err)
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err != nil {
			tt.Errorf("AuthTestAnonymous: %v.", err)
		}
	})

	t.Run("AlgNoneToken", func(tt *testing.T) {
		ci, err := cli.ConnectionInfoFromHost(&cli.Host{
			ApiEndpoint:      testListenAddr,
			ExpectedCertFile: certFile,
			AuthToken:        testAlgNoneToken,
		})
		if err != nil {
			tt.Fatalf("ConnectionInfoFromHost: %v", err)
		}

		conn, err := ci.DialGrpc()
		if err != nil {
			tt.Fatalf("DialGrpc: %v", err)
		}
		defer conn.Close()

		ssc := pb.NewSystemInfoServiceClient(conn)

		ctx := context.Background()

		if _, err = ssc.AuthTestAdmin(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestAdmin should fail.")
		}

		if _, err = ssc.AuthTestReadOnly(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestReadOnly should fail.")
		}

		if _, err = ssc.AuthTestAnonymous(ctx, &pb.AuthTestRequest{}); err == nil {
			tt.Errorf("AuthTestAnonymous should fail.")
		}
	})

	ts.Terminate()
}
