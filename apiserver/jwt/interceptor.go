package jwt

import (
	"context"
	"crypto/ecdsa"
	"strings"

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
	if pubkey == nil {
		logger.Warningf(mylog, "Authentication is disabled. Any request to the server will treated as if it were from role \"admin\".")

		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			ctx = ContextWithUserInfo(ctx, NoauthUserInfo)
			resp, err := handler(ctx, req)
			return resp, err
		}
	}

	logger.Infof(mylog, "Authentication is enabled.")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, grpc.Errorf(codes.Unauthenticated, "JwtInterceptor requires metadata.")
		}

		auths := md.Get(AuthorizationKey)
		if len(auths) == 0 {
			// ctx not modified
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

			ui, err := NewUserInfo(claims.Role, claims.Subject)
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
			}

			ctx = ContextWithUserInfo(ctx, ui)
		}

		resp, err := handler(ctx, req)
		return resp, err
	}
}
