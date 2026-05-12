package store

import "errors"

// ErrNotImplemented marks persistence paths that require a concrete backend implementation.
var ErrNotImplemented = errors.New("store: not implemented")
