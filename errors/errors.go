package errors

import (
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	errorspb "github.com/honeybbq/go-zero-errors-proto/errors"
)

// 扩展字段定义
var (
	E_Code        = errorspb.E_Code
	E_DefaultCode = errorspb.E_DefaultCode
)

const (
	// UnknownCode is unknown code for error info.
	UnknownCode = 500
	// UnknownReason is unknown reason for error info.
	UnknownReason = ""
	// SupportPackageIsVersion1 this constant should not be referenced by any other code.
	SupportPackageIsVersion1 = true
)

// Status represents the error status
type Status struct {
	Code     int32             `json:"code,omitempty"`
	Reason   string            `json:"reason,omitempty"`
	Message  string            `json:"message,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	ID       string            `json:"id,omitempty"` // 错误ID，用于追踪
}

// Error is a status error.
type Error struct {
	Status
	cause error
}

// getGoroutineID 获取当前goroutine ID
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])

	// 从stack trace中提取goroutine ID
	// stack格式: "goroutine 1 [running]:\n..."
	for i := 10; i < len(stack); i++ {
		if stack[i] == ' ' {
			if id, err := strconv.ParseUint(stack[10:i], 10, 64); err == nil {
				return id
			}
			break
		}
	}
	return 0
}

// generateRandomSuffix 生成随机后缀，避免时间戳冲突
func generateRandomSuffix() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return fmt.Sprintf("%x", buf)
}

// generateErrorID 生成包含丰富debug信息的错误ID
func generateErrorID(skip int) string {
	// 获取调用者信息
	pc, file, line, ok := runtime.Caller(skip)
	var filename, funcName, pkgName string

	if !ok {
		filename = "unknown"
		funcName = "unknown"
		pkgName = "unknown"
		line = 0
	} else {
		// 文件名
		filename = filepath.Base(file)

		// 函数信息
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fullName := fn.Name()
			// 分离包名和函数名
			if lastSlash := findLastSlash(fullName); lastSlash >= 0 {
				if lastDot := findLastDot(fullName[lastSlash:]); lastDot >= 0 {
					pkgName = fullName[:lastSlash+lastDot]
					funcName = fullName[lastSlash+lastDot+1:]
				} else {
					pkgName = "main"
					funcName = fullName[lastSlash+1:]
				}
			} else {
				pkgName = "main"
				funcName = fullName
			}
		} else {
			funcName = "unknown"
			pkgName = "unknown"
		}
	}

	// 获取各种debug信息
	timestamp := time.Now().UnixNano()
	goroutineID := getGoroutineID()
	pid := os.Getpid()
	randomSuffix := generateRandomSuffix()

	// 组合所有信息
	// 格式: pkg.func@file:line:timestamp:gid:pid:random
	info := fmt.Sprintf("%s.%s@%s:%d:%d:%d:%d:%s",
		pkgName, funcName, filename, line, timestamp, goroutineID, pid, randomSuffix)

	// Base64编码混淆
	encoded := base64.StdEncoding.EncodeToString([]byte(info))

	return encoded
}

// findLastSlash 找到最后一个斜杠的位置
func findLastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// findLastDot 找到最后一个点的位置
func findLastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// DecodeErrorID 解码错误ID，用于debug (仅在开发环境使用)
func DecodeErrorID(encodedID string) (map[string]interface{}, error) {
	decoded, err := base64.StdEncoding.DecodeString(encodedID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode error ID: %w", err)
	}

	info := string(decoded)
	// 解析格式: pkg.func@file:line:timestamp:gid:pid:random

	result := make(map[string]interface{})
	result["raw"] = info

	// 这里可以添加更详细的解析逻辑
	// 但为了简单起见，目前只返回原始信息

	return result, nil
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("error: id = %s code = %d reason = %s message = %s metadata = %v cause = %v",
			e.ID, e.Code, e.Reason, e.Message, e.Metadata, e.cause)
	}
	return fmt.Sprintf("error: code = %d reason = %s message = %s metadata = %v cause = %v",
		e.Code, e.Reason, e.Message, e.Metadata, e.cause)
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (e *Error) Unwrap() error { return e.cause }

// Is matches each error in the chain with the target value.
func (e *Error) Is(err error) bool {
	if se := new(Error); stderrors.As(err, &se) {
		return se.Code == e.Code && se.Reason == e.Reason
	}
	return false
}

// WithCause with the underlying cause of the error.
func (e *Error) WithCause(cause error) *Error {
	err := Clone(e)
	err.cause = cause
	return err
}

// WithMetadata with an MD formed by the mapping of key, value.
func (e *Error) WithMetadata(md map[string]string) *Error {
	err := Clone(e)
	err.Metadata = md
	return err
}

// WithID sets a custom error ID. If not called, a default ID will be generated.
func (e *Error) WithID(id string) *Error {
	err := Clone(e)
	err.ID = id
	return err
}

// GetID returns the error ID, generating one if it doesn't exist
func (e *Error) GetID() string {
	if e.ID == "" {
		e.ID = generateErrorID(3) // skip GetID, caller, and the method that called GetID
	}
	return e.ID
}

// GRPCStatus returns the Status represented by se.
func (e *Error) GRPCStatus() *status.Status {
	// 确保有错误ID
	if e.ID == "" {
		e.ID = generateErrorID(3)
	}

	// 将错误ID添加到metadata中传递给gRPC
	metadata := make(map[string]string)
	if e.Metadata != nil {
		for k, v := range e.Metadata {
			metadata[k] = v
		}
	}
	metadata["error_id"] = e.ID

	s, _ := status.New(ToGRPCCode(int(e.Code)), e.Message).WithDetails(&errorspb.Status{
		Code:     e.Code,
		Reason:   e.Reason,
		Message:  e.Message,
		Metadata: metadata,
	})
	return s
}

// New returns an error object for the code, reason, message.
func New(code int, reason, message string) *Error {
	return &Error{
		Status: Status{
			Code:    int32(code),
			Reason:  reason,
			Message: message,
			ID:      generateErrorID(2), // skip New and the caller
		},
	}
}

// Newf New(code, reason, fmt.Sprintf(format, a...))
func Newf(code int, reason, format string, a ...any) *Error {
	return &Error{
		Status: Status{
			Code:    int32(code),
			Reason:  reason,
			Message: fmt.Sprintf(format, a...),
			ID:      generateErrorID(2), // skip Newf and the caller
		},
	}
}

// Errorf returns an error object for the code, message and error info.
func Errorf(code int, reason, format string, a ...any) error {
	return &Error{
		Status: Status{
			Code:    int32(code),
			Reason:  reason,
			Message: fmt.Sprintf(format, a...),
			ID:      generateErrorID(2), // skip Errorf and the caller
		},
	}
}

// Clone deep clone error to a new error.
func Clone(err *Error) *Error {
	if err == nil {
		return nil
	}
	metadata := make(map[string]string, len(err.Metadata))
	for k, v := range err.Metadata {
		metadata[k] = v
	}
	return &Error{
		cause: err.cause,
		Status: Status{
			Code:     err.Code,
			Reason:   err.Reason,
			Message:  err.Message,
			Metadata: metadata,
			ID:       err.ID, // 保持原有ID
		},
	}
}

// FromError try to convert an error to *Error.
// It supports wrapped errors.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); stderrors.As(err, &se) {
		// 如果已经是我们的错误类型，确保有ID
		if se.ID == "" {
			se.ID = generateErrorID(3)
		}
		return se
	}
	gs, ok := status.FromError(err)
	if !ok {
		return &Error{
			Status: Status{
				Code:    UnknownCode,
				Reason:  UnknownReason,
				Message: err.Error(),
				ID:      generateErrorID(2),
			},
		}
	}
	ret := &Error{
		Status: Status{
			Code:    int32(ToHTTPCode(gs.Code())),
			Reason:  UnknownReason,
			Message: gs.Message(),
			ID:      generateErrorID(2),
		},
	}
	for _, detail := range gs.Details() {
		switch d := detail.(type) {
		case *errorspb.Status:
			ret.Code = d.Code
			ret.Reason = d.Reason
			ret.Message = d.Message
			ret.Metadata = d.Metadata
			// 从gRPC metadata中提取错误ID
			if d.Metadata != nil && d.Metadata["error_id"] != "" {
				ret.ID = d.Metadata["error_id"]
				// 从返回的metadata中移除error_id，避免重复
				delete(d.Metadata, "error_id")
				ret.Metadata = d.Metadata
			}
			return ret
		case *anypb.Any:
			if s := new(errorspb.Status); d.MessageIs(s) {
				_ = d.UnmarshalTo(s)
				ret.Code = s.Code
				ret.Reason = s.Reason
				ret.Message = s.Message
				ret.Metadata = s.Metadata
				// 从gRPC metadata中提取错误ID
				if s.Metadata != nil && s.Metadata["error_id"] != "" {
					ret.ID = s.Metadata["error_id"]
					// 从返回的metadata中移除error_id，避免重复
					delete(s.Metadata, "error_id")
					ret.Metadata = s.Metadata
				}
				return ret
			}
		}
	}
	return ret
}

// ID returns the error ID for a particular error.
// It supports wrapped errors.
func ID(err error) string {
	if err == nil {
		return ""
	}
	appErr := FromError(err)
	if appErr != nil {
		return appErr.GetID()
	}
	return ""
}

//
// Utility functions that match go-kratos API
//

// Code returns the http code for an error.
// It supports wrapped errors.
func Code(err error) int {
	if err == nil {
		return 200
	}
	return int(FromError(err).Code)
}

// Reason returns the reason for a particular error.
// It supports wrapped errors.
func Reason(err error) string {
	if err == nil {
		return UnknownReason
	}
	return FromError(err).Reason
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
func As(err error, target any) bool { return stderrors.As(err, target) }

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error { return stderrors.Unwrap(err) }

//
// Convenience constructors that match go-kratos API
//

// BadRequest new BadRequest error that is mapped to a 400 response.
func BadRequest(reason, message string) *Error {
	return New(400, reason, message)
}

// Unauthorized new Unauthorized error that is mapped to a 401 response.
func Unauthorized(reason, message string) *Error {
	return New(401, reason, message)
}

// Forbidden new Forbidden error that is mapped to a 403 response.
func Forbidden(reason, message string) *Error {
	return New(403, reason, message)
}

// NotFound new NotFound error that is mapped to a 404 response.
func NotFound(reason, message string) *Error {
	return New(404, reason, message)
}

// Conflict new Conflict error that is mapped to a 409 response.
func Conflict(reason, message string) *Error {
	return New(409, reason, message)
}

// InternalServer new InternalServer error that is mapped to a 500 response.
func InternalServer(reason, message string) *Error {
	return New(500, reason, message)
}

// ServiceUnavailable new ServiceUnavailable error that is mapped to an HTTP 503 response.
func ServiceUnavailable(reason, message string) *Error {
	return New(503, reason, message)
}

// GatewayTimeout new GatewayTimeout error that is mapped to an HTTP 504 response.
func GatewayTimeout(reason, message string) *Error {
	return New(504, reason, message)
}

// ClientClosed new ClientClosed error that is mapped to an HTTP 499 response.
func ClientClosed(reason, message string) *Error {
	return New(499, reason, message)
}

//
// Convenience checkers that match go-kratos API
//

// IsBadRequest determines if err is an error which indicates a BadRequest error.
// It supports wrapped errors.
func IsBadRequest(err error) bool {
	return Code(err) == 400
}

// IsUnauthorized determines if err is an error which indicates an Unauthorized error.
// It supports wrapped errors.
func IsUnauthorized(err error) bool {
	return Code(err) == 401
}

// IsForbidden determines if err is an error which indicates a Forbidden error.
// It supports wrapped errors.
func IsForbidden(err error) bool {
	return Code(err) == 403
}

// IsNotFound determines if err is an error which indicates an NotFound error.
// It supports wrapped errors.
func IsNotFound(err error) bool {
	return Code(err) == 404
}

// IsConflict determines if err is an error which indicates a Conflict error.
// It supports wrapped errors.
func IsConflict(err error) bool {
	return Code(err) == 409
}

// IsInternalServer determines if err is an error which indicates an Internal error.
// It supports wrapped errors.
func IsInternalServer(err error) bool {
	return Code(err) == 500
}

// IsServiceUnavailable determines if err is an error which indicates an Unavailable error.
// It supports wrapped errors.
func IsServiceUnavailable(err error) bool {
	return Code(err) == 503
}

// IsGatewayTimeout determines if err is an error which indicates a GatewayTimeout error.
// It supports wrapped errors.
func IsGatewayTimeout(err error) bool {
	return Code(err) == 504
}

// IsClientClosed determines if err is an error which indicates a IsClientClosed error.
// It supports wrapped errors.
func IsClientClosed(err error) bool {
	return Code(err) == 499
}

// ToGRPCCode converts an HTTP error code into the corresponding gRPC response status.
func ToGRPCCode(code int) codes.Code {
	switch code {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.Aborted
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	}
	return codes.Unknown
}

// ToHTTPCode converts a gRPC error code into the corresponding HTTP response status.
func ToHTTPCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

// TODO: This package will be complemented by a command-line tool.
// The tool will parse error definitions from .proto files (similar to protoc-gen-go-errors)
// and generate Go code (enums for reasons, helper functions like IsXXX, ErrorXXX)
// that utilizes the Error struct and mechanisms defined in this package.
