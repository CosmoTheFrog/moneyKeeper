package models

import "errors"

var (
	ErrInitDataInvalid = errors.New("init data invalid")
	ErrInitDataExpired = errors.New("init data expired")
)
