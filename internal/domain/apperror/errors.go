package apperror

import (
	"errors"
	"fmt"
)

// ErrorCode categoriza o tipo do erro para mapeamento de status HTTP.
type ErrorCode string

const (
	CodeNotFound   ErrorCode = "NOT_FOUND"
	CodeConflict   ErrorCode = "CONFLICT"
	CodeValidation ErrorCode = "VALIDATION"
	CodeInternal   ErrorCode = "INTERNAL"
)

// AppError é o tipo canônico de erro da aplicação.
type AppError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

func NotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg}
}

func Validation(msg string) *AppError {
	return &AppError{Code: CodeValidation, Message: msg}
}

func Internal(msg string, cause error) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, Cause: cause}
}

func IsNotFound(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == CodeNotFound
}

func IsConflict(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == CodeConflict
}

func IsValidation(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == CodeValidation
}
