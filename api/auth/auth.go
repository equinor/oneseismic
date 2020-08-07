package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// CheckJWT ensures that a valid JWT is provided.
func CheckJWT(rsaKeys map[string]rsa.PublicKey) context.Handler {
	return func(ctx context.Context) {
		jwtmiddleware.New(jwtmiddleware.Config{
			ValidationKeyGetter: func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() == "RS256" {
					key := rsaKeys[t.Header["kid"].(string)]
					return &key, nil
				}
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Method.Alg())
			},
		}).Serve(ctx)
	}
}

// OboJWT gets the on behalf token
func OboJWT(tokenEndpoint, clientID, clientSecret string) context.Handler {
	return func(ctx context.Context) {
		token, ok := ctx.Values().Get("jwt").(*jwt.Token)
		if !ok {
			ctx.StatusCode(http.StatusInternalServerError)
			return
		}
		data := "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer" +
			"&client_id=" + clientID +
			"&client_secret=" + clientSecret +
			"&assertion=" + token.Raw +
			"&scope=" + "https://storage.azure.com/user_impersonation" +
			"&requested_token_use=on_behalf_of"
		oboTokenResponse, err := http.Post(
			tokenEndpoint,
			"application/x-www-form-urlencoded", bytes.NewBuffer([]byte(data)))

		if err != nil {
			ctx.Values().Remove("jwt")
			golog.Errorf("could not get obo token:  %w", err)
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}
		if oboTokenResponse.StatusCode != 200 {
			ctx.Values().Remove("jwt")
			golog.Errorf(
				"could not get obo token: %v", 
				oboTokenResponse.Status, 
			)
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		// golog.Infof("obo_token: %v", oboTokenResponse)
		defer oboTokenResponse.Body.Close()
		type oboToken struct {
			AccessToken string `json:"access_token"`
		}
		obo := oboToken{}
		err = json.NewDecoder(oboTokenResponse.Body).Decode(&obo)
		if err != nil {
			golog.Warn(err)
			return
		}
		ctx.Values().Set("jwt", obo.AccessToken)
		ctx.Next()
	}
}

// Validate iss and aud in claim
func Validate(issuer, audience string) context.Handler {
	return func(ctx context.Context) {
		userToken := ctx.Values().Get("jwt")
		if userToken == nil {
			golog.Error("token missing from context")
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		user, ok := userToken.(*jwt.Token)
		if !ok {
			golog.Error("not a jwt.Token")
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		if user.Claims == nil {
			golog.Error("missing claims")
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		claims := user.Claims.(jwt.MapClaims)
		if !claims.VerifyIssuer(issuer, true) {
			golog.Errorf("invalid issuer: %v != %v", issuer, claims["iss"].(string))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		if !claims.VerifyAudience(audience, true) {
			golog.Errorf("invalid audience: %v != %v", audience, claims["aud"].(string))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		ctx.Next()
	}
}
