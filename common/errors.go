package common

import (
	"fmt"
	"net/http"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type Error struct {
	status  int
	code    string
	message string
	cause   error
}

func (this *Error) Error() string {
	return fmt.Sprintf("Error: %v, %v, %v, %v", this.status, this.code, this.message, this.cause)
}

func (this *Error) Status() int {
	return this.status
}

func (this *Error) Code() string {
	return this.code
}

func (this *Error) Message() string {
	return this.message
}

func (this *Error) Cause() error {
	return this.cause
}

func (this *Error) SetCode(code string) *Error {
	this.code = code
	return this
}

func (this *Error) SetMessage(format string, args ...interface{}) *Error {
	this.message = fmt.Sprintf(format, args...)
	return this
}

func (this *Error) SetCause(err error) *Error {
	this.cause = err
	return this
}

// FromK8sError convert a k8s error to Error
func FromK8sError(err error) *Error {
	if k8serrors.IsNotFound(err) {
		return NewNotFound().SetCause(err)
	}
	if k8serrors.IsAlreadyExists(err) || k8serrors.IsConflict(err) {
		return NewConflict().SetCause(err)

	}
	return NewInternalServerError().SetCause(err)
}

// Check following URL before add any new functions:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status

// NewBadRequest create a bad request error
func NewBadRequest() *Error {
	return &Error{
		status:  http.StatusBadRequest,
		code:    "BadRequest",
		message: "bad request",
	}
}

// NewConflict create a conflict error
func NewConflict() *Error {
	return &Error{
		status:  http.StatusConflict,
		code:    "Conflict",
		message: "conflict",
	}
}

// NewUnauthorized create a unauthorized error
func NewUnauthorized() *Error {
	return &Error{
		status:  http.StatusUnauthorized,
		code:    "Unauthorized",
		message: "unauthorized",
	}
}

// NewForbidden create a forbidden error
func NewForbidden() *Error {
	return &Error{
		status:  http.StatusForbidden,
		code:    "Forbidden",
		message: "forbidden",
	}
}

// NewNotFound create a not found error
func NewNotFound() *Error {
	return &Error{
		status:  http.StatusNotFound,
		code:    "NotFound",
		message: "not found",
	}
}

// NewMethodNotAllowed create a method not allowed error
func NewMethodNotAllowed() *Error {
	return &Error{
		status:  http.StatusMethodNotAllowed,
		code:    "MethodNotAllowed",
		message: "method not allowed",
	}
}

// NewInternalServerError create a internal server error
func NewInternalServerError() *Error {
	return &Error{
		status:  http.StatusInternalServerError,
		code:    "InternalServerError",
		message: "internal server error",
	}
}

// NewNotImplemented create a not implemented error
func NewNotImplemented() *Error {
	return &Error{
		status:  http.StatusNotImplemented,
		code:    "NotImplemented",
		message: "not implemented",
	}
}

// NewPayloadTooLarge create a payload too large error
func NewPayloadTooLarge() *Error {
	return &Error{
		status:  http.StatusRequestEntityTooLarge,
		code:    "PayloadTooLarge",
		message: "payload too large",
	}
}

// NewServiceUnavailable create a service unavailabe error
func NewServiceUnavailable() *Error {
	return &Error{
		status:  http.StatusServiceUnavailable,
		code:    "ServiceUnavailable",
		message: "service unavailable",
	}
}
