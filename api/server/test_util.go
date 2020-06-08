package server

import (
	"crypto/rand"
	"crypto/rsa"

	jwt "github.com/dgrijalva/jwt-go"
)

func mockRSAKeysJwt() (map[string]rsa.PublicKey, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	kid := "a"

	keys := make(map[string]rsa.PublicKey)
	keys[kid] = *privateKey.Public().(*rsa.PublicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{})
	token.Header["kid"] = kid
	jwt, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return keys, jwt

}
