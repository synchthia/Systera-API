package status

import (
	"errors"

	"google.golang.org/grpc/codes"
)

// ErrPlayerNotFound - When player does not exists
var ErrPlayerNotFound = &Error{
	Error: errors.New("player not found"),
	Code:  "ERR_PLAYER_NOT_FOUND",
	GrpcError: &GrpcError{
		Codes: codes.NotFound,
	},
}

// ErrPlayerAlreadyExists - When player is already exists
var ErrPlayerAlreadyExists = &Error{
	Error: errors.New("player already exists"),
	Code:  "ERR_PLAYER_ALREADY_EXISTS",
	GrpcError: &GrpcError{
		Codes: codes.AlreadyExists,
	},
}
