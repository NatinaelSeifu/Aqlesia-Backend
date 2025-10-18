package errors

import (
	"net/http"

	"github.com/joomcode/errorx"
)

var ErrorMap = map[*errorx.Type]int{
	ErrInvalidUserInput:              http.StatusBadRequest,
	ErrDataExists:                    http.StatusBadRequest,
	ErrBusinessConstraintViolation:   http.StatusBadRequest,
	ErrReadError:                     http.StatusInternalServerError,
	ErrWriteError:                    http.StatusInternalServerError,
	ErrNoRecordFound:                 http.StatusNotFound,
	ErrUnauthorized:                  http.StatusUnauthorized,
}

var (
	invalidInput = errorx.NewNamespace("validation error").ApplyModifiers(errorx.TypeModifierOmitStackTrace)
	dbError      = errorx.NewNamespace("db error")
	duplicate    = errorx.NewNamespace("duplicate").ApplyModifiers(errorx.TypeModifierOmitStackTrace)
	dataNotFound = errorx.NewNamespace("data not found").ApplyModifiers(errorx.TypeModifierOmitStackTrace)
	unauthorized = errorx.NewNamespace("unauthorized").ApplyModifiers(errorx.TypeModifierOmitStackTrace)
)

var (
	ErrInvalidUserInput            = errorx.NewType(invalidInput, "invalid user input")
	ErrBusinessConstraintViolation = errorx.NewType(invalidInput, "business constraint violation")
	ErrWriteError                  = errorx.NewType(dbError, "could not write to db")
	ErrReadError                   = errorx.NewType(dbError, "could not read data from db")
	ErrDataExists                  = errorx.NewType(duplicate, "data already exists")
	ErrNoRecordFound               = errorx.NewType(dataNotFound, "no record found")
	ErrUnauthorized                = errorx.NewType(unauthorized, "unauthorized")
)

// Error code constants for error checking
const (
	InvalidUserInput            = "invalid user input"
	BusinessConstraintViolation = "business constraint violation" 
	WriteError                  = "could not write to db"
	ReadError                   = "could not read data from db"
	DataExists                  = "data already exists"
	NoRecordFound               = "no record found"
	Unauthorized                = "unauthorized"
)

// IsErrorCode checks if the error is of a specific type
func IsErrorCode(err error, code string) bool {
	if err == nil {
		return false
	}
	// Check if it's an errorx type
	if errType, ok := err.(*errorx.Error); ok {
		return errType.Type().FullName() == code || errType.Message() == code
	}
	return false
}
