package errs

import "errors"

// Tokenizer errors
var (
	ErrInvalidToken = errors.New("invalid token")
)

// DB errors
var (
	ErrNoRows = errors.New("no rows found")
)

// Core errors
var (
	ErrNoSuchUser         = errors.New("no such user")
	ErrCouldNotSetRefresh = errors.New("could not set refresh in DB")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrWrongRefresh       = errors.New("wrong refresh token")
)
