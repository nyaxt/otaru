package jwt

import (
	"crypto/ecdsa"
	"strings"

	//jwt "gopkg.in/dgrijalva/jwt-go.v3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("jwt")

const kAuthorizationKey = "authorization"
const kBearerPrefix = "Bearer "

func UnaryServerInterceptor(pubkeys []*ecdsa.PublicKey) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			auths := md.Get(kAuthorizationKey)
			if len(auths) != 1 {
				return nil, grpc.Errorf(codes.Unauthenticated, "Invalid number of authorization values.")
			}
			auth := auths[0]

			if !strings.HasPrefix(auth, kBearerPrefix) {
				return nil, grpc.Errorf(codes.Unauthenticated, "Unsupported authroization type.")
			}
			token := auth[len(kBearerPrefix):]

			logger.Infof(mylog, "token %s", token)
		}

		resp, err := handler(ctx, req)

		return resp, err
	}
}
