package api

import (
	"fmt"
)

const (
	IllegalAccessErrorType int = 10
	NonExistingErrorType   int = 20
	InternalErrorType      int = 30
	IllegalInputErrorType  int = 40
)

//  For the extensions-part, see
//  https://spec.graphql.org/June2018/#sec-Errors
// and
//  https://github.com/graph-gophers/graphql-go (last section on page)

type IllegalAccessError struct{
	Message string
}
func NewIllegalAccessError(msg string) IllegalAccessError {
	return IllegalAccessError{Message: msg}
}
func (e IllegalAccessError) Error() string {
	return fmt.Sprintf("Illegal access")
}
func (e IllegalAccessError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"type":    IllegalAccessErrorType,
		"detail-message": e.Message,
	}
}

type NonExistingError struct{
	Message string
}
func NewNonExistingError(msg string) NonExistingError {
	return NonExistingError{Message: msg}
}
func (e NonExistingError) Error() string {
	return fmt.Sprintf("The requested object doesn't exist")
}
func (e NonExistingError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"type":    NonExistingErrorType,
		"detail-message": e.Message,
	}
}

func NewInternalError(msg string) InternalError {
	return InternalError{Message: msg}
}
type InternalError struct {
	Message string
}
func (e InternalError) Error() string {
	return fmt.Sprintf("Something unexpected happened internally. Please report this to the OneSeismic-team!")
}
func (e InternalError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"type":    InternalErrorType,
		"detail-message": e.Message,
	}
}

func NewIllegalInputError(msg string) IllegalInputError {
	return IllegalInputError{Message: msg}
}
type IllegalInputError struct {
	Message string
}
func (e IllegalInputError) Error() string {
	return fmt.Sprintf("Illegal or incomplete input")
}
func (e IllegalInputError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"type":    IllegalInputErrorType,
		"detail-message": e.Message,
	}
}
