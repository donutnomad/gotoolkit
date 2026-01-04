package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsGrpcError(t *testing.T) {
	t.Run("grpc error", func(t *testing.T) {
		grpcErr := status.Error(codes.NotFound, "resource not found")
		assert.True(t, IsGrpcError(grpcErr))
	})

	t.Run("wrapped grpc error", func(t *testing.T) {
		grpcErr := status.Error(codes.Internal, "internal error")
		wrapped := Wrap(grpcErr, "context")
		// status.FromError 会自动 Unwrap
		assert.True(t, IsGrpcError(wrapped))
	})

	t.Run("non-grpc error", func(t *testing.T) {
		regularErr := errors.New("regular error")
		assert.False(t, IsGrpcError(regularErr))
	})

	t.Run("nil error", func(t *testing.T) {
		// nil 会被 status.FromError 转换为 codes.OK
		assert.True(t, IsGrpcError(nil))
	})
}

func TestIsGrpcCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     codes.Code
		expected bool
	}{
		{
			name:     "matching code",
			err:      status.Error(codes.NotFound, "not found"),
			code:     codes.NotFound,
			expected: true,
		},
		{
			name:     "non-matching code",
			err:      status.Error(codes.NotFound, "not found"),
			code:     codes.Internal,
			expected: false,
		},
		{
			name:     "wrapped grpc error with matching code",
			err:      Wrap(status.Error(codes.PermissionDenied, "denied"), "context"),
			code:     codes.PermissionDenied,
			expected: true,
		},
		{
			name:     "non-grpc error",
			err:      errors.New("regular error"),
			code:     codes.Unknown,
			expected: false,
		},
		{
			name:     "nil error checks as OK",
			err:      nil,
			code:     codes.OK,
			expected: true,
		},
		{
			name:     "nil error not other codes",
			err:      nil,
			code:     codes.NotFound,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGrpcCode(tt.err, tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGrpcCode_AllCodes(t *testing.T) {
	// 测试所有常用的 gRPC 错误码
	testCodes := []codes.Code{
		codes.OK,
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss,
		codes.Unauthenticated,
	}

	for _, code := range testCodes {
		t.Run(code.String(), func(t *testing.T) {
			err := status.Error(code, "test error")
			assert.True(t, IsGrpcCode(err, code))

			// 确保不会匹配其他错误码
			otherCode := codes.Code((int(code) + 1) % 17)
			if otherCode != code {
				assert.False(t, IsGrpcCode(err, otherCode))
			}
		})
	}
}
