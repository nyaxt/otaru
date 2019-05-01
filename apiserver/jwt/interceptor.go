package jwt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
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

var NoJWTAuth = &JWTAuthProvider{pubkey: nil}

func NewJWTAuthProvider(pubkey *ecdsa.PublicKey) *JWTAuthProvider {
	if pubkey == nil {
		panic("Valid pubkey must be provided!")
	}
	return &JWTAuthProvider{pubkey: pubkey}
}

func NewJWTAuthProviderFromFile(path string) (*JWTAuthProvider, error) {
	var pk *ecdsa.PublicKey
	if path != "" {
		keytext, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("Failed to load ECDSA public key file %q: %v", path, err)
		}

		pk, err = jwt.ParseECPublicKeyFromPEM(keytext)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse ECDSA public key %q: %v", path, err)
		}
	}

	return NewJWTAuthProvider(pk), nil
}

func TokenStringFromAuthHeader(auth string) (string, error) {
	if auth == "" {
		return "", nil
	}
	if !strings.HasPrefix(auth, BearerPrefix) {
		return "", errors.New("Unsupported authroization type.")
	}
	return auth[len(BearerPrefix):], nil
}

func (p *JWTAuthProvider) IsEnabled() bool {
	return p.pubkey != nil
}

func (p *JWTAuthProvider) UserInfoFromTokenString(tokenString string) (*UserInfo, error) {
	if !p.IsEnabled() {
		panic("NoJWTAuth cannot extract UserInfoFromTokenString.")
	}

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

type jwtTokenKey struct{}

func ContextWithJWTTokenString(ctx context.Context, tokenstr string) context.Context {
	return context.WithValue(ctx, jwtTokenKey{}, tokenstr)
}

func JWTTokenStringFromContext(ctx context.Context) string {
	tokenstr, ok := ctx.Value(jwtTokenKey{}).(string)
	if !ok {
		return ""
	}
	return tokenstr
}

func (p *JWTAuthProvider) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if !p.IsEnabled() {
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

			tokenstr, err := TokenStringFromAuthHeader(auths[0])
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
			}

			ui, err := p.UserInfoFromTokenString(tokenstr)
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
			}

			ctx = ContextWithJWTTokenString(ctx, tokenstr)
			ctx = ContextWithUserInfo(ctx, ui)
		}

		resp, err := handler(ctx, req)
		return resp, err
	}
}
