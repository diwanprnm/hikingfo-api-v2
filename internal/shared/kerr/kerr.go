// Package kerr defines the uniform error used across contexts and the HTTP
// boundary. Every application service returns a *Error so the interfaces layer
// maps it 1:1 onto the API's `{ "error": { code, message, fields } }` shape
// (contracts/api.md → Conventions → Error shape).
package kerr

import (
	"errors"
	"fmt"
)

// Code enumerates the stable machine-readable error codes of the API.
type Code string

const (
	CodeValidation   Code = "validation_failed"
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeRateLimited  Code = "rate_limited"
	CodeInternal     Code = "internal"
)

// Error is the uniform domain/application error.
type Error struct {
	Code    Code
	Message string
	Fields  map[string]string // field -> problem, for 422-style detail
	Err     error             // wrapped cause, never serialised
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// WithField attaches one field-level problem (immutable style: returns a copy).
func (e *Error) WithField(field, problem string) *Error {
	cp := *e
	cp.Fields = make(map[string]string, len(e.Fields)+1)
	for k, v := range e.Fields {
		cp.Fields[k] = v
	}
	cp.Fields[field] = problem
	return &cp
}

// Constructors. Message is the user-facing default (English chrome; the
// frontend owns locale display); detail is reserved for logs.

func Validation(msg string) *Error   { return &Error{Code: CodeValidation, Message: msg} }
func Unauthorized(msg string) *Error { return &Error{Code: CodeUnauthorized, Message: msg} }
func Forbidden(msg string) *Error    { return &Error{Code: CodeForbidden, Message: msg} }
func NotFound(msg string) *Error     { return &Error{Code: CodeNotFound, Message: msg} }
func Conflict(msg string) *Error     { return &Error{Code: CodeConflict, Message: msg} }
func RateLimited(msg string) *Error  { return &Error{Code: CodeRateLimited, Message: msg} }
func Internal(msg string) *Error     { return &Error{Code: CodeInternal, Message: msg} }

// Wrapping constructors that carry a cause.

func WrapValidation(msg string, cause error) *Error {
	return &Error{Code: CodeValidation, Message: msg, Err: cause}
}
func WrapInternal(msg string, cause error) *Error {
	return &Error{Code: CodeInternal, Message: msg, Err: cause}
}
func WrapNotFound(msg string, cause error) *Error {
	return &Error{Code: CodeNotFound, Message: msg, Err: cause}
}
func WrapConflict(msg string, cause error) *Error {
	return &Error{Code: CodeConflict, Message: msg, Err: cause}
}

// From converts any error into a *Error, mapping wrapped sentinel causes and
// preserving an existing *Error. Unknown errors become CodeInternal.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var ke *Error
	if errors.As(err, &ke) {
		return ke
	}
	return WrapInternal("internal error", err)
}
