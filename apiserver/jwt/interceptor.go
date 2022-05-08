package jwt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const (
	AuthorizationKey = "authorization"
	BearerPrefix     = "Bearer "
)

type JWTAuthProvider struct {
	Disabled bool
}

func UserInfoFromClientCert(cert *x509.Certificate) UserInfo {
	cn := cert.Subject.CommonName

	a := strings.SplitN(cn, " ", 2)
	rolestr := a[0]
	user := rolestr
	if len(a) > 1 {
		user = a[1]
	}

	return UserInfo{Role: RoleFromStr(rolestr), User: user}
}

var ErrZeroVerifiedChains = errors.New("JWTAuthProvider could not find a client cert.")
var ErrZeroVerifiedChains2 = errors.New("JWTAuthProvider requires len(VerifiedChains[0]) > 0.")

func UserInfoFromTLSConnectionState(tcs *tls.ConnectionState) (UserInfo, error) {
	vcs := tcs.VerifiedChains
	if len(vcs) == 0 {
		return AnonymousUserInfo, ErrZeroVerifiedChains
	}
	vc := vcs[0]
	if len(vc) == 0 {
		return AnonymousUserInfo, ErrZeroVerifiedChains2
	}

	return UserInfoFromClientCert(vc[0]), nil
}

func (p JWTAuthProvider) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if p.Disabled {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			ctx = ContextWithUserInfo(ctx, NoauthUserInfo)
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, grpc.Errorf(codes.Unauthenticated, "JWTAuthProvider requires metadata.")
		}
		if p.AuthInfo == nil {
			return nil, grpc.Errorf(codes.Unauthenticated, "JWTAuthProvider requires grpc Peer with AuthInfo.")
		}
		ti, ok := p.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return nil, grpc.Errorf(codes.Unauthenticated, "JWTAuthProvider requires grpc Peer with credentails.TLSInfo.")
		}

		ui, err := UserInfoFromTLSConnectionState(&ti.State)
		if err != nil {
			return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
		}

		ctx = ContextWithUserInfo(ctx, ui)
		return handler(ctx, req)
	}
}
