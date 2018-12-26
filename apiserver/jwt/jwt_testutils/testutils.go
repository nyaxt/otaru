package jwt_testutils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"time"

	jwt "gopkg.in/dgrijalva/jwt-go.v3"

	ojwt "github.com/nyaxt/otaru/apiserver/jwt"
)

var Key *ecdsa.PrivateKey
var Pubkey *ecdsa.PublicKey
var AdminToken string
var ReadOnlyToken string
var AlgNoneToken string

var JWTAuthProvider *ojwt.JWTAuthProvider

func init() {
	var err error
	Key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("ecdsa.GenerateKey")
	}
	Pubkey = &Key.PublicKey

	now := time.Now()

	claims := ojwt.Claims{
		Role: ojwt.RoleAdmin.String(),
		StandardClaims: jwt.StandardClaims{
			Audience:  ojwt.OtaruAudience,
			ExpiresAt: (now.Add(time.Hour)).Unix(),
			Issuer:    "auth_test",
			NotBefore: now.Unix(),
			Subject:   "auth_test",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	AdminToken, err = token.SignedString(Key)
	if err != nil {
		panic("AdminToken")
	}

	claims.Role = ojwt.RoleReadOnly.String()
	token = jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	ReadOnlyToken, err = token.SignedString(Key)
	if err != nil {
		panic("ReadOnlyToken")
	}

	token = jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	AlgNoneToken, err = token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		panic("AlgNoneToken")
	}

	JWTAuthProvider = ojwt.NewJWTAuthProvider(Pubkey)
}
