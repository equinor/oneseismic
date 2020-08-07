package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

type oidConfig struct {
	JwksURI string `json:"jwks_uri"`
	Issuer string `json:"issuer"`
	TokenEndpoint string `json:"token_endpoint"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

var httpGet func(string) (*http.Response, error)

func init() {
	configClient := &http.Client{Timeout: 10 * time.Second}
	httpGet = configClient.Get
}

func getJSON(url string, target interface{}) error {
	r, err := httpGet(url)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf(
			"Json fetch error %s on %s",
			r.Status,
			url)

	}

	return json.NewDecoder(r.Body).Decode(target)
}

func fromB64(b64 string) (big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(b64)
	bi := big.Int{}
	if err != nil {
		return bi, fmt.Errorf("decoding B64 failed: %w", err)
	}

	bi.SetBytes(b)
	return bi, nil
}

// OpenIDConfig has the config we need to enable auth
type OpenIDConfig struct {
	Jwks map[string]rsa.PublicKey
	Issuer string
	TokenEndpoint string
}

// GetOidConfig gets OpenIDConfig from a well-known openid-configuration url
func GetOidConfig(authserver string) (*OpenIDConfig, error) {
	oidc := oidConfig{}
	err := getJSON(authserver, &oidc)
	if err != nil {
		return nil, fmt.Errorf("getting oidc failed: %w", err)
	}

	keyList := jwks{}
	err = getJSON(oidc.JwksURI, &keyList)
	if err != nil {
		return nil, fmt.Errorf("getting jwks failed: %w", err)
	}

	keys := make(map[string]rsa.PublicKey)

	for _, key := range keyList.Keys {

		if key.Kty == "RSA" {

			e, err := fromB64(key.E)
			if err != nil {
				return nil, fmt.Errorf("big int from  E: %w", err)
			}
			n, err := fromB64(key.N)
			if err != nil {
				return nil, fmt.Errorf("big int from  N: %w", err)
			}

			keys[key.Kid] = rsa.PublicKey{N: &n, E: int(e.Int64())}

		}
	}

	return &OpenIDConfig{
		Jwks:          keys,
		Issuer:        oidc.Issuer,
		TokenEndpoint: oidc.TokenEndpoint,
	}, nil
}
