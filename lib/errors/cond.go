package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func IsGrpcError(err error) bool {
	if _, ok := status.FromError(err); ok {
		return true
	}
	return false
}

// IsGrpcCode 判断一个错误是否是特定的 gRPC 错误码
func IsGrpcCode(err error, code codes.Code) bool {
	if st, ok := status.FromError(err); ok {
		return st.Code() == code
	}
	return false
}
