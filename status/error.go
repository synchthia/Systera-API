package status

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GrpcError - Custom grpc error
type GrpcError struct {
	Codes codes.Code
}

// Error - Custom error struct
type Error struct {
	Error     error      `json:"error"`
	Code      string     `json:"code"`
	GrpcError *GrpcError `json:"grpc_error,omitempty"`
}

func (e *Error) ToGrpcError() *status.Status {
    s := status.New(e.GrpcError.Codes, e.Error.Error())
    s.WithDetails(&errdetails.ErrorInfo{
        Reason: e.Error.Error(),
        Metadata: map[string]string{
            "code": e.Code,
        },
    })
    return s
}
