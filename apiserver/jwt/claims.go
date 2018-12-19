package jwt

import (
	jwt "gopkg.in/dgrijalva/jwt-go.v3"
)

const OtaruAudience = "otaru"

type Claims struct {
	Role string `json:"role"`
	jwt.StandardClaims
}

var otaruRoleSet = map[string]struct{}{
	"admin":     struct{}{},
	"readonly":  struct{}{},
	"anonymous": struct{}{},
}

func IsValidRole(role string) bool {
	_, valid := otaruRoleSet[role]
	return valid
}
