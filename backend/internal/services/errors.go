package services

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
)
