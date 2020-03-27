package middleware

import (
	"fmt"
	"net/url"

	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12/context"
)

type OAuth2Option struct {
	AuthServer *url.URL
	Audience   string
	Issuer     string
	APISecret  []byte
}

func Oauth2(oauthOpt OAuth2Option) (context.Handler, error) {
	sigKeySet, err := GetOIDCKeySet(oauthOpt.AuthServer)
	if err != nil {
		return nil, fmt.Errorf("could not get keyset: %w", err)
	}

	rsaJWTHandler := jwtmiddleware.New(jwtmiddleware.Config{
		ValidationKeyGetter: func(t *jwt.Token) (interface{}, error) {

			if t.Method.Alg() != "RS256" {
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
			}
			return sigKeySet[t.Header["kid"].(string)], nil

		},
		ContextKey:    "user-jwt",
		SigningMethod: jwt.SigningMethodRS256,
	})

	onRS256Pass := func(ctx context.Context, err error) {

		if err == nil || err.Error() == "unexpected jwt signing method=RS256" {
			return
		}
		jwtmiddleware.OnError(ctx, err)
	}
	hmacJWTHandler := jwtmiddleware.New(jwtmiddleware.Config{
		ValidationKeyGetter: func(t *jwt.Token) (interface{}, error) {

			if t.Method.Alg() != "HS256" {
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
			}
			return oauthOpt.APISecret, nil
		},
		ContextKey:    "service-jwt",
		SigningMethod: jwt.SigningMethodHS256,
		ErrorHandler:  onRS256Pass,
	})

	if len(oauthOpt.Issuer) == 0 {
		oauthOpt.Issuer = oauthOpt.AuthServer.String()
	}

	auth := func(ctx context.Context) {
		hmacJWTHandler.Serve(ctx)
		serviceToken := ctx.Values().Get("service-jwt")
		if serviceToken == nil {
			rsaJWTHandler.Serve(ctx)
		}

	}

	return auth, nil
}
