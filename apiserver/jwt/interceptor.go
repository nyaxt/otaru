package jwt

import (
	"crypto/ecdsa"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	jwt "gopkg.in/dgrijalva/jwt-go.v3"

	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("jwt")

const (
	AuthorizationKey = "authorization"
	BearerPrefix     = "Bearer "
)

func UnaryServerInterceptor(pubkey *ecdsa.PublicKey) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, grpc.Errorf(codes.Unauthenticated, "JwtInterceptor requires metadata.")
		}

		auths := md.Get(AuthorizationKey)
		if len(auths) == 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, "user", "anonymous", "role", "anonymous")
		} else {
			if len(auths) != 1 {
				return nil, grpc.Errorf(codes.Unauthenticated, "Invalid number of authorization values.")
			}
			auth := auths[0]

			if !strings.HasPrefix(auth, BearerPrefix) {
				return nil, grpc.Errorf(codes.Unauthenticated, "Unsupported authroization type.")
			}
			tokenString := auth[len(BearerPrefix):]

			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
					return nil, grpc.Errorf(codes.Unauthenticated, "Unexpected signing method: %v", token.Header["alg"])
				}

				return pubkey, nil
			})
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "Failed to parse jwt token")
			}

			claims, ok := token.Claims.(*Claims)
			if !ok || !token.Valid {
				return nil, grpc.Errorf(codes.Unauthenticated, "Failed to validate jwt token")
			}

			if !claims.VerifyAudience(OtaruAudience, true) {
				return nil, grpc.Errorf(codes.Unauthenticated, "Failed to validate jwt token audience.")
			}

			if claims.NotBefore == 0 {
				return nil, grpc.Errorf(codes.Unauthenticated, "The jwt token is missing \"nbf\" claim.")
			}
			if claims.ExpiresAt == 0 {
				return nil, grpc.Errorf(codes.Unauthenticated, "The jwt token is missing \"exp\" claim.")
			}
			if err := claims.Valid(); err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "The jwt token is not valid at this time.")
			}

			if claims.Subject == "" {
				return nil, grpc.Errorf(codes.Unauthenticated, "The jwt token is missing \"sub\" claim.")
			}

			if !IsValidRole(claims.Role) {
				return nil, grpc.Errorf(codes.Unauthenticated, "The jwt token has unknown role claim.")
			}

			ctx = metadata.AppendToOutgoingContext(ctx, "user", claims.Subject, "role", claims.Role)
			// logger.Infof(mylog, "user: %q, role: %q", claims.Subject, claims.Role)
		}

		resp, err := handler(ctx, req)

		return resp, err
	}
}
