package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func verifyIssuerAudience(
	issuer   string,
	audience string,
	token    *jwt.Token,
) error {
	claims := token.Claims.(jwt.MapClaims)
	if !claims.VerifyAudience(audience, false) {
		return fmt.Errorf("Invalid audience; wanted %s, got %s", audience, claims["aud"])
	}

	if !claims.VerifyIssuer(issuer, false) {
		return fmt.Errorf("Invalid issuer; wanted %s, got %s", issuer, claims["iss"])
	}

	return nil
}

func validateKey(
	keys  map[string]rsa.PublicKey,
	token *jwt.Token,
) (interface {}, error) {
	keyID, ok := token.Header["kid"];
	if !ok {
		return nil, fmt.Errorf("'kid' not in JWT.Header")
	}
	key, ok := keys[keyID.(string)];
	if !ok {
		return nil, fmt.Errorf("key not recognized; id = %s", keyID)
	}
	return &key, nil
}

/*
 * Make a function that validates the contents of the JWT token in the
 * Authorization header.
 *
 * The implementation itself is heavily influenced by how the JWT middlware and
 * gin works, so there's not too much wiggle room here.
 *
 * Notes
 * -----
 * The keys and issuer params are obtained through the OpenID connect protocol.
 * 
 * The audience claim is specific to this application, i.e. the application
 * performing requests on-behalf-of its clients.
 */
func ValidateJWT(
	keys     map[string]rsa.PublicKey,
	issuer   string,
	audience string,
) gin.HandlerFunc {
	auth := jwtmiddleware.New(jwtmiddleware.Options {
		SigningMethod: jwt.SigningMethodRS256,
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			err := verifyIssuerAudience(issuer, audience, token)
			if err != nil {
				log.Printf("%v", err)
				return nil, err
			}
			key, err := validateKey(keys, token)
			if err != nil {
				log.Printf("%v", err)
			}
			return key, err
		},
	})

	return func (ctx *gin.Context) {
		if err := auth.CheckJWT(ctx.Writer, ctx.Request); err != nil {
			log.Printf("checkJWT() failed: %v", err)
			ctx.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

/*
 * Check that the authorization header is well-formatted
 */
func checkAuthorizationHeader(authorization string) error {
	// TODO: ensure that the CheckJWT function checks the authorization header
	// suffienctly well
	if authorization == "" {
		return fmt.Errorf("Request without JWT header, but passed validation")
	}

	if !strings.HasPrefix(authorization, "Bearer") {
		return fmt.Errorf("Authorization not a Bearer token")
	}

	return nil
}

func fetchOnBehalfToken(
	host      string,
	id        string,
	secret    string,
	assertion string,
) (*http.Response, error) {

	/*
	 * TODO: Could take gin.Context and abort directly, to since error handling
	 * regardless boils down to just killing the ongoing request
	 */

	/* 
	 * The host is the token endpoint from the OpenID Connect config, which in
	 * turn is fetch'd from the auth server.
	 *
	 * The assertion is what's in the JWT token in the Authorization header:
	 * Authorization: Bearer $assertion
	 */
	 parameters := []string {
		"grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer",
		"client_id=" + id,
		"client_secret=" + secret,
		"assertion=" + assertion,
		"scope=" + "https://storage.azure.com/user_impersonation",
		"requested_token_use=on_behalf_of",
	}
	request := strings.Join(parameters[:], "&")

	return http.Post(
		host,
		"application/x-www-form-urlencoded",
		bytes.NewBuffer([]byte(request)),
	)
}

type oboToken struct {
	AccessToken string `json:"access_token"`
}

func (o *oboToken) UnmarshalJSON(b []byte) error {
	type OnBehalfOfToken oboToken
	aux := OnBehalfOfToken {}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}

	if aux.AccessToken == "" {
		return fmt.Errorf("missing field 'access_token'")
	}

	o.AccessToken = aux.AccessToken
	return nil
}

/*
 * Re-write the authorization header, and perform requests to Azure on behalf
 * of the caller
 *
 * TODO: this should probably be broken up a bit more and more thoroughly
 * tested
 *
 * https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-on-behalf-of-flow
 */
func OnBehalfOf(
	loginAddr    string,
	clientID     string,
	clientSecret string,
) gin.HandlerFunc {

	return func (ctx *gin.Context) {
		// "Authorization: Bearer $token"
		token := ctx.GetHeader("Authorization")
		if err := checkAuthorizationHeader(token); err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			// TODO: should the authorization header itself be logged?
			log.Printf("%v", err)
			return
		}

		response, err := fetchOnBehalfToken(
			loginAddr,
			clientID,
			clientSecret,
			strings.Split(token, " ")[1],
		)
		if err != nil {
			log.Printf("Could not perform request for obo token: %v", err)
			// InternalServerError?
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			log.Printf("Request for obo token failed: %v", response.Status)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		obo := oboToken{}
		err = json.NewDecoder(response.Body).Decode(&obo)
		if err != nil {
			log.Printf("Unable to decode obo token: %v", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.Set("OBOJWT", obo.AccessToken)
	}
}
