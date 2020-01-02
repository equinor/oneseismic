package claims

import (
	"crypto/subtle"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

type Middleware struct {
	validators []ClaimsValidator
	audience   string
	issuer     string
}

type ClaimsValidator func(jwt.MapClaims) error

func New(audience, issuer string, validators ...ClaimsValidator) *Middleware {
	m := &Middleware{
		validators: validators,
	}

	if len(audience) > 0 {
		m.audience = audience
		m.validators = append(m.validators, m.verifyAud)
	}
	if len(issuer) > 0 {
		m.issuer = issuer
		m.validators = append(m.validators, m.verifyIss)
	}
	return m
}

func (m *Middleware) Validate(ctx context.Context) {
	serviceToken := ctx.Values().Get("service-jwt")
	if serviceToken != nil {
		l.LogI("Have service token")
		ctx.Next()
		return
	}

	userToken := ctx.Values().Get("user-jwt")
	if userToken == nil {
		l.LogE("Get token from context", fmt.Errorf("No user token"))
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
	validationErrors := make([]error, 0)
	claims := user.Claims.(jwt.MapClaims)
	if err := claims.Valid(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	for _, validator := range m.validators {
		err := validator(claims)
		if err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			l.LogE("Validate claims", e)
		}
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.StopExecution()
		return
	}

	ctx.Next()

}

func (m *Middleware) verifyAud(c jwt.MapClaims) error {
	if c["aud"] == nil {
		return nil
	}

	if subtle.ConstantTimeCompare([]byte(m.audience), []byte(c["aud"].(string))) == 0 {
		return fmt.Errorf("Invalid audience %s", c["aud"])
	}

	return nil

}

func (m *Middleware) verifyIss(c jwt.MapClaims) error {

	if subtle.ConstantTimeCompare([]byte(m.issuer), []byte(c["iss"].(string))) == 0 {
		return fmt.Errorf("Invalid issuer %s", c["iss"])
	}
	return nil
}
