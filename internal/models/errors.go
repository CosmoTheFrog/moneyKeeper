package models

import "errors"

var (
	ErrInitDataInvalid = errors.New("init data invalid")
	ErrInitDataExpired = errors.New("init data expired")

	ErrNotFound        = errors.New("not found")
	ErrBaseCurrencySet = errors.New("base currency already set")
	ErrInvalidAccount  = errors.New("invalid account")
	ErrInvalidTimezone = errors.New("invalid timezone")
)
