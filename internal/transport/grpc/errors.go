package grpcserver

import "errors"

// Shared error sentinels used across handlers.
var (
	errCmdFull           = errors.New("app command channel full; server is busy")
	errEmptyPath         = errors.New("path must not be empty")
	errPathTraversal     = errors.New("path must be under data/systems/")
	errNoJumpTargets     = errors.New("names must not be empty")
	errInvalidDimensions = errors.New("width and height must both be > 0")
	errInvalidOrbit      = errors.New("name must not be empty, speed must be non-zero, orbits must be > 0")
)
