package common

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ErrMsgUnauthenticated  string = "sign in require"
	ErrMsgPermissionDenied string = "you don't have permission to access"
	ErrMsgInternal         string = "Internal Server Error"
)

var GRPCErrUnauthenticated = status.Error(codes.Unauthenticated, ErrMsgUnauthenticated)
var GRPCERRPermissionDenied = status.Error(codes.PermissionDenied, ErrMsgPermissionDenied)
var GRPCErrInternal = status.Error(codes.Internal, ErrMsgInternal)
