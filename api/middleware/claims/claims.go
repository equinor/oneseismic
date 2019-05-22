package claims

import (
	"crypto/subtle"
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type Middleware struct {
	validators []ClaimsValidator
	audience   string
	issuer     string
}

type ClaimsValidator func(jwt.Claims) error

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
	user := ctx.Values().Get("jwt").(*jwt.Token)
	validationErrors := make([]error, 0)
	stdClaims := user.Claims.(jwt.StandardClaims)
	if err := stdClaims.Valid(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	for _, validator := range m.validators {
		err := validator(user.Claims)
		if err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		for e := range validationErrors {
			log.Println("Claims invalid", e)
		}
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.StopExecution()
		return
	}

	ctx.Next()

}

func (m *Middleware) verifyAud(c jwt.Claims) error {

	sc, ok := c.(jwt.StandardClaims)
	if !ok {
		return fmt.Errorf("Claims are not standard")
	}
	if subtle.ConstantTimeCompare([]byte(m.audience), []byte(sc.Audience)) != 0 {
		return fmt.Errorf("Invalid audience %s", sc.Audience)
	}

	return nil

}

func (m *Middleware) verifyIss(c jwt.Claims) error {
	sc, ok := c.(jwt.StandardClaims)
	if !ok {
		return fmt.Errorf("Claims are not standard")
	}
	if subtle.ConstantTimeCompare([]byte(m.issuer), []byte(sc.Issuer)) != 0 {
		return fmt.Errorf("Invalid issuer %s", sc.Issuer)
	}
	return nil
}
