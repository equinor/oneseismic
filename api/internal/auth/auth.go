package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"sync"

	"github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"github.com/gin-gonic/gin"
)

/*
 * The status error is an internal error type that provides a (suggested) http
 * error code and a (suggested) log message. Virtually all errors should result
 * in aborting the request and setting the appropriate HTTP return code, and
 * this has to be communicated from helper functions somehow.
 *
 * Without a custom and predicatable error type, then functions that call
 * helpers would have to know intimately all error paths and possible error
 * types when writing log messages and setting status codes, which defeats the
 * purpose of the helper functions. This is not designed to be an exported.
 */
type statusError struct {
	status  int
	message string
}

func (s *statusError) Error() string {
	return s.message
}

type Tokens interface {
	GetOnbehalf(auth string) (string, error)
}

type TokenFetch struct {
	loginAddr    string
	clientID     string
	clientSecret string
	cache        sync.Map
}

func NewTokens(
	loginAddr    string,
	clientID     string,
	clientSecret string,
) Tokens {
	return &TokenFetch {
		loginAddr:    loginAddr,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (t *TokenFetch) GetOnbehalf(token string) (string, error) {
	if cached, found := t.cache.Load(token); found {
		return cached.(string), nil
	}

	if err := checkAuthorizationHeader(token); err != nil {
		// TODO: should the authorization header itself be logged?
		return "", &statusError {
			status: http.StatusBadRequest,
			message: fmt.Sprintf("%v", err),
		}
	}

	/*
	 * TODO: this could use some more tests to make sure that error paths are
	 * properly taken and errors are properly set. Unfortunately, it's mostly
	 * I/O and invoking third-party parsing, so testing is a right pain and an
	 * underwhelming gain, so it's put off until later.
	 */
	response, err := fetchOnBehalfToken(
		t.loginAddr,
		t.clientID,
		t.clientSecret,
		strings.Split(token, " ")[1],
	)
	if err != nil {
		return "", &statusError {
			status:  http.StatusUnauthorized,
			message: fmt.Sprintf("Request for obo token failed: %v", err),
		}
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", &statusError {
			status:  http.StatusUnauthorized,
			message: response.Status,
		}
	}

	obo := oboToken{}
	err = json.NewDecoder(response.Body).Decode(&obo)
	if err != nil {
		return "", &statusError {
			status: http.StatusInternalServerError,
			message: fmt.Sprintf("Token decoding failed: %v", err),
		}
	}
	t.cache.Store(token, obo.AccessToken)
	return obo.AccessToken, nil
}

/*
 * A stupid helper that automates the inspection of the (unexported!) error
 * type and setting the right status code. Logging is left to the caller, but
 * the statusError supports the Error interface, so this should be perfectly
 * fine.
 */
func AbortContextFromToken(ctx *gin.Context, err error) {
	switch e := err.(type) {
	case *statusError:
		ctx.AbortWithStatus(e.status)
	default:
		ctx.AbortWithStatus(http.StatusInternalServerError)
	}
}

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
 * The Keyring is the concept of making, signing, and parsing tokens that
 * ensure that a result resource is only available to the one who requested it
 * [1]. It's based on a pre-shared key which can be randomly generated on
 * application startup, and given as environment or argument to whatever
 * service that needs it.
 *
 * [1] providing the token is not shared or leaked, but this is a problem with
 *     all token-based access
 */
type Keyring struct {
	key []byte
}

/*
 * A stupid constructor function, really only to hide the key field and maybe
 * at some point do validation.
 */
func MakeKeyring(key []byte) Keyring {
	return Keyring {
		key: key,
	}
}

/*
 * Sign with the default timeout - in practice, this is the only sign function
 * there should be a need for, and gives a single point for updates, bugfixes
 * and reasonable configuration.
 */
func (k *Keyring) Sign(pid string) (string, error) {
	expiration := time.Now().Add(5 * time.Minute)
	return k.SignWithTimeout(pid, expiration)
}

/*
 * Sign, but with a custom timeout. This function is largely an implementation
 * detail, and is intended for testing (e.g. creating already-expired tokens).
 * However, it might provide useful as an escape hatch should a non-default
 * timeout be needed.
 */
func (r *Keyring) SignWithTimeout(
	pid string,
	exp time.Time,
) (string, error) {
	claims := &jwt.MapClaims {
		"pid": pid,
		"exp": exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(r.key)
}

/*
 * Validate a key - if this function returns nil, the token is valid for
 * accessing the result and status of the process $pid.
 */
func (r *Keyring) Validate(tokenstr string, pid string) error {
	/*
	 * The jwt library is built around having multiple keys available, and
	 * choosing the right one from the token header (see the key-id (kid) logic
	 * in this module). This is not used currently, and it's only the
	 * pre-shared key in play. This may certainly change in the future, in
	 * which case it's the keyfunc that's responsible for picking out and
	 * returning the right key.
	 */
	keyfunc := func (t *jwt.Token) (interface {}, error) {
		return r.key, nil
	}
	token, err := jwt.Parse(tokenstr, keyfunc)

	if err != nil {
		return err
	}

	if token.Valid {
		/*
		 * The docs [1] are a bit unclear, but it seems reasonable to assume
		 * that when parsing a token, the returned token.Claims (an interface)
		 * is always of MapClaims. This has to be cast accordingly to look up
		 * the oneseismic specific key/value "pid". This works at least for
		 * now, but will break spectacularly should jwt-go change this, in
		 * which case the parsing approach must be revisited.
		 *
		 * [1] https://godoc.org/github.com/dgrijalva/jwt-go
		 */
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			msg := "expected 'claims' of type jwt.MapClaims; was %T"
			return fmt.Errorf(msg, claims)
		}

		/*
		 * The token is valid if the pid in the token matches the pid of the
		 * request, and the token is signed. From our implementation's point of
		 * view, this really boils down to a string comparison.
		 *
		 * The token itself is signed, so a token that did not originate in the
		 * oneseismic service will have a signature mismatch. Since the
		 * *content* of the token contributes to the signature, it is not
		 * possible to use a valid token for a different process to both pass
		 * the signature check *and* the string comparison.
		 */
		tokenpid := claims["pid"]
		if tokenpid == pid {
			return nil
		}
		return fmt.Errorf("token with invalid pid; got %v", tokenpid)
	}

	return fmt.Errorf("Keyring.Validate fell through; This is a logic error")
}

/*
 * Middleware to auth the token returned by /query, which must be included with
 * requests to get access to /result. Any request in the /result family must
 * check the token and it should be independent of the operation, so it makes a
 * perfect fit for the middleware. This also makes it reasonably easy to test.
 *
 * The procedure boils down to:
 * - Get Authorization header
 * - Decode it and check signature
 * - Check that the pid in the token matches the pid for the request
 *
 * That way, only the one who made the request can query the status or get the
 * result.
 */
func ResultAuth(keyring *Keyring) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		pid := ctx.Param("pid")
		authorization := ctx.GetHeader("Authorization")
		if authorization == "" {
			log.Printf("%s No Authorization header", pid)
			/*
			 * MDN docs
			 * --------
			 * Although the HTTP standard specifies "unauthorized",
			 * semantically this response means "unauthenticated". That is, the
			 * client must authenticate itself to get the requested response.
			 *
			 * https://developer.mozilla.org/en-US/docs/Web/HTTP/Status
			 */
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token := ""
		_, err := fmt.Sscanf(authorization, "Bearer %s", &token)
		if err != nil {
			log.Printf(
				"%s malformed header Authorization; was %s",
				pid,
				authorization,
			)
			/*
			 * Malformed authorization header - not quite sure if this is
			 * Unauthorized, BadRequest or some other status code. Unauthorized
			 * seems the most appropriate based on a few quick searches, so use
			 * that until a good authorative source can be provided.
			 */
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		err = keyring.Validate(token, pid)
		if err != nil {
			log.Printf("%s %v", pid, err)
			ctx.AbortWithStatus(http.StatusForbidden)
		}
	}
}
