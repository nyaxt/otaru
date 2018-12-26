package jwt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
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

type JWTAuthProvider struct {
	pubkey *ecdsa.PublicKey
}

func NewJWTAuthProvider(pubkey *ecdsa.PublicKey) *JWTAuthProvider {
	if pubkey == nil {
		logger.Infof(mylog, "Authentication is disabled. Any request to the server will treated as if it were from role \"admin\".")
		return &JWTAuthProvider{pubkey: nil}
	}

	logger.Infof(mylog, "Authentication is enabled.")
	return &JWTAuthProvider{pubkey: pubkey}
}

func (p *JWTAuthProvider) UserInfoFromAuthHeader(auth string) (*UserInfo, error) {
	if p.pubkey == nil && auth == "" {
		return NoauthUserInfo, nil
	}

	if !strings.HasPrefix(auth, BearerPrefix) {
		return nil, errors.New("Unsupported authroization type.")
	}
	tokenString := auth[len(BearerPrefix):]

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return p.pubkey, nil
	})
	if err != nil {
		return nil, errors.New("Failed to parse jwt token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("Failed to validate jwt token")
	}

	if !claims.VerifyAudience(OtaruAudience, true) {
		return nil, errors.New("Failed to validate jwt token audience.")
	}

	if claims.NotBefore == 0 {
		return nil, errors.New("The jwt token is missing \"nbf\" claim.")
	}
	if claims.ExpiresAt == 0 {
		return nil, errors.New("The jwt token is missing \"exp\" claim.")
	}
	if err := claims.Valid(); err != nil {
		return nil, errors.New("The jwt token is not valid at this time.")
	}

	if claims.Subject == "" {
		return nil, errors.New("The jwt token is missing \"sub\" claim.")
	}

	ui, err := NewUserInfo(claims.Role, claims.Subject)
	if err != nil {
		return nil, err
	}
	return ui, nil
}

func (p *JWTAuthProvider) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if p.pubkey == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			ctx = ContextWithUserInfo(ctx, NoauthUserInfo)
			resp, err := handler(ctx, req)
			return resp, err
		}
	}

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

			ui, err := p.UserInfoFromAuthHeader(auths[0])
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
			}

			ctx = ContextWithUserInfo(ctx, ui)
		}

		resp, err := handler(ctx, req)
		return resp, err
	}
}
