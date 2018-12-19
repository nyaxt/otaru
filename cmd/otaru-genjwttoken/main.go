package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	jwt "gopkg.in/dgrijalva/jwt-go.v3"

	ojwt "github.com/nyaxt/otaru/apiserver/jwt"
)

const IssuerStr = "otaru-genjwttoken"

var (
	flagPrivKey = flag.String("privkey", "jwt.key", "ECDSA private key file to be used for signing.")
	flagSubject = flag.String("subject", "user", "The user name to be encoded as a subject of the jwt token.")
	flagExpire  = flag.Duration("expire", 1*time.Hour, "The expire time of the jwt token.")
	flagRole    = flag.String("role", "", "The otaru role to assign the jwt token.")
)

func run() error {
	keyfile := *flagPrivKey
	role := *flagRole
	if !ojwt.IsValidRoleStr(role) {
		return fmt.Errorf("Invalid role %q.", role)
	}

	keytext, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return fmt.Errorf("Failed to load ECDSA private key file %q: %v", keyfile, err)
	}

	key, err := jwt.ParseECPrivateKeyFromPEM(keytext)
	if err != nil {
		return fmt.Errorf("Failed to parse ECDSA private key %q: %v", keyfile, err)
	}

	now := time.Now()

	claims := ojwt.Claims{
		Role: role,
		StandardClaims: jwt.StandardClaims{
			Audience:  ojwt.OtaruAudience,
			ExpiresAt: (now.Add(*flagExpire)).Unix(),
			Issuer:    IssuerStr,
			NotBefore: now.Unix(),
			Subject:   *flagSubject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	ss, err := token.SignedString(key)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", ss)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
