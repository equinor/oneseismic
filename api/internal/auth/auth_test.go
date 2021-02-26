package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/form3tech-oss/jwt-go"
	"github.com/gin-gonic/gin"
)

func TestAudienceIssuerValidator(t *testing.T) {
	tests := []struct {
		claims jwt.MapClaims
		prefix string
	}{
		{
			claims: jwt.MapClaims{
				"iss": "valid_issuer",
				"aud": "valid_audience",
			},
			prefix: "",
		},
		{
			claims: jwt.MapClaims{
				"iss": "invalid_issuer",
				"aud": "valid_audience",
			},
			prefix: "Invalid issuer",
		},
		{
			claims: jwt.MapClaims{
				"iss": "valid_issuer",
				"aud": "invalid_audience",
			},
			prefix: "Invalid audience",
		},
	}
	hs256 := jwt.GetSigningMethod("HS256")
	for _, tt := range tests {
		token := jwt.NewWithClaims(hs256, tt.claims)
		err := verifyIssuerAudience("valid_issuer", "valid_audience", token)

		if err == nil {
			if tt.prefix != "" {
				t.Errorf("Expected success; got %v", err)
			} else {
				continue
			}
		}

		if !strings.HasPrefix(err.Error(), tt.prefix) {
			t.Errorf("Expected prefix %v; got %v", tt.prefix, err)
		}
	}
}

func TestValidateKeyFailsMissingKey(t *testing.T) {
	keys := map[string]rsa.PublicKey {
		"some-key": rsa.PublicKey {},
	}
	token := &jwt.Token {
		Header: make(map[string]interface{}),
	}

	_, err := validateKey(keys, token)
	expected := "'kid' not in JWT.Header"

	if err == nil {
		t.Errorf("Expected validate to fail; got %v", err)
	}
	if err.Error() != expected {
		t.Errorf("Expected \"%s\"; got %v", expected, err)
	}
}

func TestValidateKeyFailsUnknownKey(t *testing.T) {
	keys := map[string]rsa.PublicKey {
		"some-key": rsa.PublicKey {},
	}
	token := &jwt.Token {
		Header: map[string]interface{} {
			"kid": "other-key",
		},
	}

	_, err := validateKey(keys, token)
	prefix := "key not recognized"
	if err == nil {
		t.Errorf("Expected validate to fail; got %v", err)
	}
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Errorf("Expected error message prefix \"%s\"; got %v", prefix, err)
	}
}

func TestOBOTokenMissingFields(t *testing.T) {
	doc := "{}"
	obo := oboToken {}
	err := json.Unmarshal([]byte(doc), &obo)
	if err == nil {
		t.Errorf("expected missing-field error, got nil; in %s", doc)
	}
}

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

func TestOnbehalfBadTokenGivesBadRequest(t *testing.T) {
	tokens := NewTokens("", "", "")

	empty := ""
	_, err := tokens.GetOnbehalf(empty)
	status := err.(*statusError).status
	if status != http.StatusBadRequest {
		t.Errorf("Empty token should set status 400 BadRequest")
	}

	nobearer := "valid-but-malformed"
	_, err = tokens.GetOnbehalf(nobearer)
	status = err.(*statusError).status
	if status != http.StatusBadRequest {
		t.Errorf("Malformed token should status 400 BadRequest")
	}
}
