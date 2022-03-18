package auth

import (
	"context"
	"crypto/rsa"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	
	"github.com/golang-jwt/jwt"
	"github.com/gin-gonic/gin"
)

func TestTokenSignRoundTrip(t *testing.T) {
	key := []byte("pre-shared-key")
	keyring := MakeKeyring(key)

	pid := "pid"
	token, err := keyring.Sign(pid)
	if err != nil {
		t.Fatalf("Error creating token; %v", err)
	}

	err = keyring.Validate(token, pid)
	if err != nil {
		keyfunc := func (tok *jwt.Token) (interface {}, error) {
			return key, nil
		}
		token, _ := jwt.Parse(token, keyfunc)
		msg := "Expected valid token (error = %v); token was %v"
		t.Errorf(msg, err, token.Claims)
	}
}

func TestExpiredTokenIsInvalid(t *testing.T) {
	key := []byte("pre-shared-key")
	keyring := MakeKeyring(key)

	pid := "pid"
	exp := time.Now().Add(-5 * time.Minute)
	token, err := keyring.SignWithTimeout(pid, exp)
	if err != nil {
		t.Fatalf("Error creating token; %v", err)
	}

	err = keyring.Validate(token, pid)
	if err == nil {
		t.Errorf("Expected expired token to be invalid, but Validate succeded")
	}
}

func TestTokenInvalidPid(t *testing.T) {
	key := []byte("pre-shared-key")
	keyring := MakeKeyring(key)

	pid := "pid"
	token, err := keyring.Sign(pid)
	if err != nil {
		t.Fatalf("Error creating token; %v", err)
	}

	err = keyring.Validate(token, "different-pid")
	if err == nil {
		t.Errorf("Expected expired token to be invalid, but Validate succeded")
	}
}

func TestValidTokenInvalidSignature(t *testing.T) {
	pid := "pid"
	/*
	 * The signature generation could maybe be done by writing the token
	 * generation by hand, to protect against bugs in the keyring
	 * implementation
	 */
	keyringA := MakeKeyring([]byte("pre-shared-key"))
	token, err := keyringA.Sign(pid)
	if err != nil {
		t.Fatalf("Error creating token; %v", err)
	}

	keyringB := MakeKeyring([]byte("pre-shared-diff-key"))
	err = keyringB.Validate(token, pid)
	if err == nil {
		t.Errorf("Expected expired token to be invalid, but Validate succeded")
	}
}

func TestResultAuthTokens(t *testing.T) {
	keyring := MakeKeyring([]byte("psk"))
	good, err := keyring.Sign("pid")
	if err != nil {
		t.Fatalf("%v", err)
	}

	tokens := map[string]int {
		"nil":                          http.StatusUnauthorized,
		"":                             http.StatusUnauthorized,
		"sans-token-type":              http.StatusUnauthorized,
		fmt.Sprintf("Bad %s", good):    http.StatusUnauthorized,
		"Bearer bad-key":               http.StatusForbidden,
		fmt.Sprintf("Bearer %s", good): http.StatusOK,
	}

	authfn := ResultAuth(&keyring)
	for token, expected := range tokens {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)

		r.GET("/result/:pid", authfn)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/result/pid", nil)
		/*
		 * Use the string "nil" to add the case where the Authorization header
		 * is omitted
		 */
		if token != "nil" {
			ctx.Request.Header.Add("Authorization", token)
		}

		r.ServeHTTP(w, ctx.Request)
		if w.Result().StatusCode != expected {
			msg := "Got %v; want %d %s"
			t.Errorf(msg, w.Result().Status, expected, http.StatusText(expected))
		}
	}
}

func TestJWTValidation(t *testing.T) {
	hs256key := []byte("secret")
	
	/* Key function for the HS256 algorithm */
	hs256KeyFunc := func (context.Context) (interface{}, error) {
		return hs256key, nil
	}
	
	rs256PrivKey, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		log.Fatal("failed to generate key")
	}

	/* Key function for the RS256 algoritm */
	rs256KeyFunc := func (context.Context) (interface{}, error) {
		return &rs256PrivKey.PublicKey, nil
	}

	/* A stupid helper to make the creation of the testcases a bit cleaner */
	withClaims := func(alg jwt.SigningMethod, key interface{}, claims jwt.MapClaims) (string) {
		token, err := jwt.NewWithClaims(alg, claims).SignedString(key)
		if err != nil {
			log.Fatalf("Invalid token, %v", err)
		}

		return token
	}
	
	type KeyFunc func(context.Context) (interface{}, error)

	testcases := []struct{
		name     string
		expected int
		keyFunc  KeyFunc
		token    string

	}{
		{
			name:     "No token",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    "nil",
		},
		{
			name:     "Empty token",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    "",
		},
		{
			name:     "Bad token",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    "bad token",
		},
		{
			name:     "Invalid algorithm",
			expected: http.StatusUnauthorized,
			keyFunc:  hs256KeyFunc,
			token:    withClaims(jwt.SigningMethodHS256, hs256key, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Read"},
			}),
		},
		{
			name:     "Invalid 'aud' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "invalid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Read"},
			}),
		},
		{
			name:     "Invalid 'iss' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "invalid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Read"},
			}),
		},
		{
			name:     "Invalid 'exp' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute * -1).Unix(),
				"nbf"   : time.Now().Add(time.Minute * -2).Unix(),
				"iat"   : time.Now().Add(time.Minute * -2).Unix(),
				"roles" : []string{"Read"},
			}),
		},
		{
			name:     "Invalid 'nbf' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  2).Unix(),
				"nbf"   : time.Now().Add(time.Minute *  1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Read"},
			}),
		},
		{
			name:     "Missing 'role' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
			}),
		},
		{
			name:     "Invalid 'role' claim",
			expected: http.StatusUnauthorized,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Invalid role"},
			}),
		},
		{
			name:     "Valid JWT",
			expected: http.StatusOK,
			keyFunc:  rs256KeyFunc,
			token:    withClaims(jwt.SigningMethodRS256, rs256PrivKey, jwt.MapClaims{
				"iss"   : "valid issuer",
				"aud"   : "valid audience",
				"exp"   : time.Now().Add(time.Minute *  1).Unix(),
				"nbf"   : time.Now().Add(time.Second * -1).Unix(),
				"iat"   : time.Now().Add(time.Second * -1).Unix(),
				"roles" : []string{"Read"},
			}),
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)

		tokenValidation := JWTvalidation(
			"valid issuer",
			"valid audience",
			testcase.keyFunc,
		)

		r.GET("/graphql", tokenValidation)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/graphql", nil)

		/*
		* Use the string "nil" to add the case where the Authorization header
		* is omitted
		*/
		if testcase.token != "nil" {
			ctx.Request.Header.Add(
				"Authorization",
				fmt.Sprintf("Bearer %s", testcase.token),
			)
		}

		r.ServeHTTP(w, ctx.Request)
		if w.Result().StatusCode != testcase.expected {
			msg := "Got %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expected,
				http.StatusText(testcase.expected),
				testcase.name,
			)
		}
	}
}
