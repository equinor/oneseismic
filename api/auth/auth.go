package auth

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	l "github.com/equinor/oneseismic/api/logger"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// CheckJWT ensures that a valid JWT is provided.
// Expects a bearer token in the Authorization header.
// Supports RS256 and H256.
func CheckJWT(sigKeySet map[string]interface{}, apiSecret []byte) context.Handler {
	return func(ctx context.Context) {
		jwtmiddleware.New(jwtmiddleware.Config{
			ValidationKeyGetter: func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() == "RS256" {
					return sigKeySet[t.Header["kid"].(string)], nil
				}
				if t.Method.Alg() == "HS256" {
					return apiSecret, nil
				}
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Method.Alg())
			},
		}).Serve(ctx)
	}
}

// ValidateClaims validates `aud` and `iss` claims which is not checked by CheckJWT
func ValidateClaims(audience, issuer string) context.Handler {
	return func(ctx context.Context) {
		userToken := ctx.Values().Get("jwt")
		if userToken == nil {
			l.LogE("Get token from context", fmt.Errorf("missing token"))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		user, ok := userToken.(*jwt.Token)
		if !ok || user.Claims == nil {
			l.LogE("Check user claims", fmt.Errorf("No claims"))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		claims := user.Claims.(jwt.MapClaims)
		if !claims.VerifyIssuer(issuer, false) {
			l.LogE("invalid issuer", fmt.Errorf(claims["iss"].(string)))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
		}

		ctx.Next()
	}
}
