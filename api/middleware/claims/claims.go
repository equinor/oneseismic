package claims

import (
	"log"

	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type Middleware struct {
	validators []ClaimsValidator
}

type ClaimsValidator func(jwt.Claims) error

func New(validators ...ClaimsValidator) *Middleware {
	m := &Middleware{
		validators: validators,
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
