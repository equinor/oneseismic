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

type openIDConfig struct {
	JwksURI string `json:"jwks_uri"`
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

func getJwksURI(authserver string) (string, error) {
	oidcConf := openIDConfig{}
	err := getJSON(authserver, &oidcConf)
	if err != nil {
		return "", fmt.Errorf("fetching oidc config failed: %w", err)
	}

	return oidcConf.JwksURI, nil
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

// GetRSAKeys gets a map of kid with rsa.PublicKey
func GetRSAKeys(authserver string) (map[string]rsa.PublicKey, error) {
	jwksURI, err := getJwksURI(authserver)
	if err != nil {
		return nil, fmt.Errorf("getting jwks_uri failed: %w", err)
	}

	keyList := jwks{}
	err = getJSON(jwksURI, &keyList)
	if err != nil {
		return nil, fmt.Errorf("fetching jwks failed: %w", err)
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

	return keys, nil

}
