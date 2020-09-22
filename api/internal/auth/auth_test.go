package auth

import (
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
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
