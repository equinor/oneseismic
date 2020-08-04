package auth

import (
	"crypto/rsa"
	"fmt"

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

func ValidateIssuer(issuer string) context.Handler {
	return func(ctx context.Context) {
		userToken := ctx.Values().Get("jwt")
		if userToken == nil {
			golog.Error("Get token from context", fmt.Errorf("missing token"))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		user, ok := userToken.(*jwt.Token)
		if !ok {
			golog.Error("Type assertion", fmt.Errorf("not a jwt.Token"))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		if user.Claims == nil {
			golog.Error("Check claims", fmt.Errorf("nil Claims"))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		claims := user.Claims.(jwt.MapClaims)
		if !claims.VerifyIssuer(issuer, true) {
			golog.Error("invalid issuer", fmt.Errorf(claims["iss"].(string)))
			ctx.StatusCode(iris.StatusUnauthorized)
			ctx.StopExecution()
			return
		}

		ctx.Next()
	}
}
