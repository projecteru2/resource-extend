package types

import "github.com/cockroachdb/errors"

var (
	ErrInvalidGPUMap     = errors.New("invalid gpu map")
	ErrInvalidGPUProduct = errors.New("invalid gpu product")
)
