package jwt

import (
	jwt "gopkg.in/dgrijalva/jwt-go.v3"
)

const OtaruAudience = "otaru"

type Claims struct {
	Role string `json:"role"`
	jwt.StandardClaims
}

type Role int

const (
	RoleAnonymous Role = iota
	RoleReadOnly
	RoleAdmin
)

var strToRole = map[string]Role{
	"anonymous": RoleAnonymous,
	"readonly":  RoleReadOnly,
	"admin":     RoleAdmin,
}

var roleToStr = map[Role]string{
	RoleAnonymous: "anonymous",
	RoleReadOnly:  "readonly",
	RoleAdmin:     "admin",
}

func (r Role) String() string {
	return roleToStr[r]
}
