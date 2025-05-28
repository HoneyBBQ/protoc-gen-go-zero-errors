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
	"strings"
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
func getGoroutineID() (result uint64) {
	// 添加 panic 恢复机制
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时返回 0
			result = 0
		}
	}()

	var buf [32]byte // 减小缓冲区大小，通常goroutine ID不会很长
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])

	// 从stack trace中提取goroutine ID
	// stack格式: "goroutine 1 [running]:\n..."
	start := 10 // "goroutine " 的长度
	if start >= len(stack) {
		return 0
	}

	end := start
	for end < len(stack) && stack[end] != ' ' {
		end++
	}

	if end > start {
		if id, err := strconv.ParseUint(stack[start:end], 10, 64); err == nil {
			return id
		}
	}
	return 0
}

// generateRandomSuffix 生成随机后缀，避免时间戳冲突
func generateRandomSuffix() (result string) {
	// 添加 panic 恢复机制
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时返回简单的时间戳
			result = fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)
		}
	}()

	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		return fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return fmt.Sprintf("%x", buf)
}

// generateErrorID 生成包含丰富debug信息的错误ID
func generateErrorID(skip int) string {
	// 添加 panic 恢复机制
	defer func() {
		if r := recover(); r != nil {
			// 如果发生 panic，记录并返回一个简单的错误ID
			// 这里不能使用日志，因为可能导致循环调用
		}
	}()

	// 使用内部函数尝试生成完整的错误ID
	if id := tryGenerateErrorID(skip + 1); id != "" {
		return id
	}

	// 如果内部函数失败，返回备用ID
	return generateFallbackErrorID()
}

// tryGenerateErrorID 尝试生成错误ID，如果失败返回空字符串
func tryGenerateErrorID(skip int) (result string) {
	// 添加 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			result = ""
		}
	}()

	return generateErrorIDInternal(skip)
}

// generateErrorIDInternal 内部实现，包含实际的ID生成逻辑
func generateErrorIDInternal(skip int) string {
	// 完整版本 - 包含详细信息
	// 获取调用者信息
	pc, file, line, ok := runtime.Caller(skip)
	var filename, funcName string

	if !ok {
		filename = "unknown"
		funcName = "unknown"
		line = 0
	} else {
		// 文件名 - 只保留文件名，不要完整路径
		filename = filepath.Base(file)

		// 函数信息 - 简化处理
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fullName := fn.Name()
			// 只保留函数名部分，去掉包路径
			if lastDot := findLastDot(fullName); lastDot >= 0 && lastDot < len(fullName)-1 {
				funcName = fullName[lastDot+1:]
			} else if lastDot == len(fullName)-1 {
				// 如果点在最后，使用完整名称
				funcName = fullName
			} else {
				funcName = fullName
			}
		} else {
			funcName = "unknown"
		}
	}

	// 获取关键debug信息
	timestamp := time.Now().UnixNano()
	goroutineID := getGoroutineID()
	pid := os.Getpid()
	randomSuffix := generateRandomSuffix()

	// 使用更高效的字符串构建 - 简化格式
	// 格式: func@file:line:timestamp:gid:pid:random
	var builder strings.Builder
	builder.Grow(128) // 预分配容量

	builder.WriteString(funcName)
	builder.WriteByte('@')
	builder.WriteString(filename)
	builder.WriteByte(':')
	builder.WriteString(strconv.Itoa(line))
	builder.WriteByte(':')
	builder.WriteString(strconv.FormatInt(timestamp, 10))
	builder.WriteByte(':')
	builder.WriteString(strconv.FormatUint(goroutineID, 10))
	builder.WriteByte(':')
	builder.WriteString(strconv.Itoa(pid))
	builder.WriteByte(':')
	builder.WriteString(randomSuffix)

	// Base64编码
	return base64.StdEncoding.EncodeToString([]byte(builder.String()))
}

// generateFallbackErrorID 生成一个简单的备用错误ID
func generateFallbackErrorID() string {
	// 使用最基本的信息生成ID，避免复杂操作
	timestamp := time.Now().UnixNano()
	pid := os.Getpid()

	// 使用简单的随机字节，避免复杂操作
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes) // crypto/rand.Read 不会返回错误
	randomNum := int64(randomBytes[0])<<24 | int64(randomBytes[1])<<16 | int64(randomBytes[2])<<8 | int64(randomBytes[3])

	// 格式: fallback:timestamp:pid:random
	fallbackID := fmt.Sprintf("fallback:%d:%d:%d", timestamp, pid, randomNum)
	return base64.StdEncoding.EncodeToString([]byte(fallbackID))
}

// findLastSlash 找到最后一个斜杠的位置
func findLastSlash(s string) int {
	return strings.LastIndex(s, "/")
}

// findLastDot 找到最后一个点的位置
func findLastDot(s string) int {
	return strings.LastIndex(s, ".")
}

// ErrorIDInfo 错误ID解码后的结构化信息
type ErrorIDInfo struct {
	Function      string `json:"function"`       // 函数名
	File          string `json:"file"`           // 文件名
	Line          int    `json:"line"`           // 行号
	Timestamp     int64  `json:"timestamp"`      // 纳秒时间戳
	GoroutineID   uint64 `json:"goroutine_id"`   // Goroutine ID
	ProcessID     int    `json:"process_id"`     // 进程ID
	RandomSuffix  string `json:"random_suffix"`  // 随机后缀
	TimeFormatted string `json:"time_formatted"` // 格式化的时间
	Raw           string `json:"raw"`            // 原始解码信息
}

// DecodeErrorID 解码错误ID，返回结构化信息
func DecodeErrorID(encodedID string) (*ErrorIDInfo, error) {
	decoded, err := base64.StdEncoding.DecodeString(encodedID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode error ID: %w", err)
	}

	raw := string(decoded)
	info := &ErrorIDInfo{Raw: raw}

	// 解析格式: func@file:line:timestamp:gid:pid:random
	parts := strings.Split(raw, ":")
	if len(parts) < 6 {
		return info, fmt.Errorf("invalid error ID format, expected at least 6 parts, got %d", len(parts))
	}

	// 解析函数名和文件名 (func@file 格式)
	funcFilePart := parts[0]
	if atIndex := strings.LastIndex(funcFilePart, "@"); atIndex >= 0 {
		info.Function = funcFilePart[:atIndex]
		info.File = funcFilePart[atIndex+1:]
	} else {
		info.Function = "unknown"
		info.File = funcFilePart
	}

	// 解析行号
	if line, err := strconv.Atoi(parts[1]); err == nil {
		info.Line = line
	}

	// 解析时间戳
	if timestamp, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
		info.Timestamp = timestamp
		// 格式化时间
		info.TimeFormatted = time.Unix(0, timestamp).Format("2006-01-02 15:04:05.000")
	}

	// 解析 Goroutine ID
	if gid, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
		info.GoroutineID = gid
	}

	// 解析进程ID
	if pid, err := strconv.Atoi(parts[4]); err == nil {
		info.ProcessID = pid
	}

	// 随机后缀
	if len(parts) > 5 {
		info.RandomSuffix = parts[5]
	}

	return info, nil
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

// IsClientError 检查是否为客户端错误
func (e *Error) IsClientError() bool {
	return e.Code >= 400 && e.Code < 500
}

// IsServerError 检查是否为服务器错误
func (e *Error) IsServerError() bool {
	return e.Code >= 500
}

// IsAuthError 检查是否为认证/授权错误
func (e *Error) IsAuthError() bool {
	return e.Code == 401 || e.Code == 403
}
