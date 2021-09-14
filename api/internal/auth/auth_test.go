package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/form3tech-oss/jwt-go"
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
