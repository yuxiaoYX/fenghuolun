package neta

import "errors"

var (
	ErrRefreshUnverified = errors.New("neta refresh HTTP is not verified in HAR")
	ErrUpstream          = errors.New("neta upstream")
	ErrDecode            = errors.New("neta decode")
	ErrTokenInvalid      = errors.New("neta token invalid")
)
