package interceptor

import (
	"context"
	"log"

	"github.com/honeybbq/protoc-gen-go-zero-errors/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerErrorInterceptor returns a new unary server interceptor that converts
// application-specific errors into gRPC errors using the coreerrors package.
func UnaryServerErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			// Attempt to convert any error to our *Error type
			// FromError is expected to handle nil, *Error already, and other error types.
			// If err is already a gRPC status, FromError should ideally parse it back.
			// If FromError cannot handle a specific type gracefully and returns a generic internal error,
			// that will then be converted to a gRPC status.
			appErr := errors.FromError(err)
			if appErr != nil { // Should always be non-nil if err was non-nil, as FromError creates a default
				// 确保错误有ID并记录日志
				errorID := appErr.GetID()
				log.Printf("gRPC unary error [ID: %s]: %v", errorID, err)

				return resp, appErr.GRPCStatus().Err()
			}
			// Fallback for any unexpected scenario where appErr might be nil despite err being non-nil
			// or if err was not convertible in a structured way by FromError.
			// This path should ideally not be hit if FromError is robust.
			log.Printf("unhandled error type in UnaryServerErrorInterceptor: %T, value: %v", err, err)
			return resp, status.Error(codes.Internal, err.Error()) // Default to gRPC internal error
		}
		return resp, err
	}
}

// TODO: Implement StreamServerErrorInterceptor
// StreamServerErrorInterceptor returns a new stream server interceptor that converts errors.
// This is more complex as it needs to wrap the grpc.ServerStream and intercept errors
// from RecvMsg, SendMsg, and the handler's return value.
// For a simpler first pass, it might only handle the error returned by the stream handler itself.

// Example of a simplified stream interceptor that only handles the handler's final error:
func StreamServerErrorInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss) // Call the original handler
		if err != nil {
			appErr := errors.FromError(err)
			if appErr != nil {
				// 确保错误有ID并记录日志
				errorID := appErr.GetID()
				log.Printf("gRPC stream error [ID: %s]: %v", errorID, err)

				return appErr.GRPCStatus().Err()
			}
			// Fallback
			log.Printf("unhandled error type in StreamServerErrorInterceptor: %T, value: %v", err, err)
			return status.Error(codes.Internal, err.Error()) // Default to gRPC internal error
		}
		return err
	}
}
