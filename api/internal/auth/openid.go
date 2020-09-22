package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
)

/*
 * The go net/http module only provides an implementation, and not an interface
 * for it's HTTP Get() and response. To make testing feasible without having to
 * spin up a server, implement a tiny interface that net/http implements, and
 * just substitute that in the tests.
 */
type HttpClient interface {
	Get(string) (*http.Response, error)
}

/*
 * The expected responses and objects in the OpenID Connect protocol
 *
 * https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-protocols-oidc
 */
type openIDConfig struct {
	JwksURI       string `json:"jwks_uri"`
	Issuer        string `json:"issuer"`
	TokenEndpoint string `json:"token_endpoint"`
}

/*
 * JSON Web Key https://tools.ietf.org/html/rfc7517
 */
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

/*
 * Custom unmarshal functions so that missing or malformed fields turns into
 * errors.
 *
 * The alias type is necessary to avoid going into an infinite loop of
 * unmarshaling the same type, but unfortunately it can make the error message
 * bad:
 *     cannot unmarshal [...] Go struct field JWKSAlias.keys of type []auth.jwk
 *
 * The unmarshalling does a poor job of key/values missing, and values with a
 * default value (e.g. the empty string), but for the application this
 * distinction is somewhat uninteresting as it pretty much always mean that not
 * enough parameters are available to properly configure everything.
 *
 * http://choly.ca/post/go-json-marshalling/
 */

func (cfg *openIDConfig) UnmarshalJSON(b []byte) error {
	type openIDconfig openIDConfig
	aux := openIDconfig {}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}

	if aux.JwksURI == "" {
		return fmt.Errorf("missing field 'jwks_uri'")
	}
	if aux.Issuer == "" {
		return fmt.Errorf("missing field 'issuer'")
	}
	if aux.TokenEndpoint == "" {
		return fmt.Errorf("missing field 'token_endpoint'")
	}

	cfg.JwksURI       = aux.JwksURI
	cfg.Issuer        = aux.Issuer
	cfg.TokenEndpoint = aux.TokenEndpoint
	return nil
}

func (j *jwk) UnmarshalJSON(b []byte) error {
	type JSONWebKey jwk
	aux := JSONWebKey {}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}

	if aux.Kty == "" {
		return fmt.Errorf("missing field 'kty'")
	}
	if aux.Kid == "" {
		return fmt.Errorf("missing field 'kid'")
	}
	if aux.N == "" {
		return fmt.Errorf("missing field 'n'")
	}
	if aux.E == "" {
		return fmt.Errorf("missing field 'e'")
	}
	j.Kty = aux.Kty
	j.Kid = aux.Kid
	j.N = aux.N
	j.E = aux.E
	return nil
}

func (j *jwks) UnmarshalJSON(b []byte) error {
	type JSONWebKeySet jwks
	aux := JSONWebKeySet {}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}

	if aux.Keys == nil {
		return fmt.Errorf("missing field 'keys'")
	}

	if len(aux.Keys) == 0 {
		return fmt.Errorf("found field 'keys', but the list is empty")
	}

	j.Keys = aux.Keys
	return nil
}

/*
 * Helper function that's the combination of a HTTP GET and json.Decode(), to
 * address a tiny amount of boilerplate
 */
func getJSON(c HttpClient, url string, target interface{}) error {
	r, err := c.Get(url)
	if err != nil {
		return fmt.Errorf("Could not perform HTTP GET: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf("HTTP GET %s failed %s", url, r.Status)
	}

	return json.NewDecoder(r.Body).Decode(target)
}

func fromB64(b64 string) (big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(b64)
	bi := big.Int{}
	if err != nil {
		return bi, fmt.Errorf("B64 decoding failed: %w", err)
	}

	bi.SetBytes(b)
	return bi, nil
}

/*
 * Public configuration struct with the variables necessary to auth
 */
type OpenIDConfig struct {
	Jwks          map[string]rsa.PublicKey
	Issuer        string
	TokenEndpoint string
}

func getWebKeySet(c HttpClient, url string) ([]jwk, error) {
	doc := jwks {}
	err := getJSON(c, url, &doc)
	if err != nil {
		return nil, err
	}
	return doc.Keys, nil
}

type noRSAKeys struct {}

func (e *noRSAKeys) Error() string {
	return "OpenIDConfig without any RSA keys"
}

/* 
 * Get the Open ID config from a well-known URL
 *
 * > OpenID Connect defines a discovery mechanism, called OpenID Connect
 * > Discovery, where an OpenID server publishes its metadata at a well-known
 * > URL, typically https://server.com/.well-known/openid-configuration [1]
 *
 * The implementation only supports RSA keys, and only those with n and e [2],
 * which may not be all responses the protocol specifies. Keys that don't meet
 * the expecations of this function will be skipped, and if the response
 * contains no viable keys, a noRSAKeys error will be returned.
 *
 * [1] https://swagger.io/docs/specification/authentication/openid-connect-discovery
 * [2] https://tools.ietf.org/html/rfc7517#section-9.3
 */
func GetOpenIDConfig(c HttpClient, authserver string) (*OpenIDConfig, error) {
	oidc := openIDConfig{}
	err := getJSON(c, authserver, &oidc)
	if err != nil {
		return nil, fmt.Errorf("Getting OpenID config: %w", err)
	}

	keyset, err := getWebKeySet(c, oidc.JwksURI)
	if err != nil {
		return nil, fmt.Errorf("Getting keyset: %w", err)
	}

	/*
	 * The behaviour here can be a bit wonky - getting on-behalf-of tokens will
	 * certainly fail if there are no RSA keys, but arguably the function
	 * succeeds with a good response even without any RSA keys in the key set
	 */
	keys := make(map[string]rsa.PublicKey)
	for _, key := range keyset {
		if key.Kty == "RSA" {
			e, err := fromB64(key.E)
			/* 
			 * Decoding errors means the key is probably broken, so skip it and
			 * look for other viable keys
			 */
			if err != nil {
				log.Printf("Key.E (id = %s): %v", key.Kid, err)
				continue
			}
			n, err := fromB64(key.N)
			if err != nil {
				log.Printf("Key.N (id = %s): %v", key.Kid, err)
				continue
			}

			keys[key.Kid] = rsa.PublicKey{
				N: &n,
				E: int(e.Int64()),
			}
		}
	}

	err = nil
	if len(keys) == 0 {
		err = &noRSAKeys{}
		log.Printf("Keyset: %v", keyset)
	}

	return &OpenIDConfig {
		Jwks:          keys,
		Issuer:        oidc.Issuer,
		TokenEndpoint: oidc.TokenEndpoint,
	}, err
}
