package errors

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestErrorID(t *testing.T) {
	// 测试基本的错误ID生成
	err := New(400, "TEST_ERROR", "这是一个测试错误")

	t.Log(err.ID)
	if err.ID == "" {
		t.Error("错误ID不应该为空")
	}

	// 测试ID函数
	errorID := ID(err)
	if errorID == "" {
		t.Error("ID函数返回的错误ID不应该为空")
	}

	if errorID != err.ID {
		t.Error("ID函数返回的值应该与错误对象中的ID一致")
	}
}

func TestErrorIDUniqueness(t *testing.T) {
	// 测试错误ID的唯一性
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		err := New(400, "TEST", "测试错误唯一性")
		if ids[err.ID] {
			t.Errorf("发现重复的错误ID: %s", err.ID)
		}
		ids[err.ID] = true

		// 稍微延迟一下确保时间戳不同
		time.Sleep(time.Nanosecond)
	}
}

func TestErrorIDDecoding(t *testing.T) {
	// 测试错误ID解码
	err := New(404, "NOT_FOUND", "资源未找到")

	debugInfo, decodeErr := DecodeErrorID(err.ID)
	if decodeErr != nil {
		t.Errorf("解码错误ID失败: %v", decodeErr)
	}

	if debugInfo["raw"] == "" {
		t.Error("解码后的原始信息不应该为空")
	}

	// 验证解码信息包含预期的组件
	rawInfo := debugInfo["raw"].(string)
	if !strings.Contains(rawInfo, "errors_test.go") {
		t.Errorf("解码信息应该包含文件名，实际: %s", rawInfo)
	}

	if !strings.Contains(rawInfo, "TestErrorIDDecoding") {
		t.Errorf("解码信息应该包含函数名，实际: %s", rawInfo)
	}
}

func TestErrorIDWithConvenienceFunctions(t *testing.T) {
	// 测试便利函数是否正确生成错误ID
	testCases := []struct {
		name     string
		createFn func() *Error
		code     int32
	}{
		{"BadRequest", func() *Error { return BadRequest("BAD_REQ", "无效请求") }, 400},
		{"Unauthorized", func() *Error { return Unauthorized("UNAUTH", "未授权") }, 401},
		{"Forbidden", func() *Error { return Forbidden("FORBIDDEN", "禁止访问") }, 403},
		{"NotFound", func() *Error { return NotFound("NOT_FOUND", "未找到") }, 404},
		{"Conflict", func() *Error { return Conflict("CONFLICT", "冲突") }, 409},
		{"InternalServer", func() *Error { return InternalServer("INTERNAL", "内部错误") }, 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.createFn()

			if err.ID == "" {
				t.Errorf("%s 错误应该有错误ID", tc.name)
			}

			if err.Code != tc.code {
				t.Errorf("%s 错误码应该是 %d，实际: %d", tc.name, tc.code, err.Code)
			}

			// 验证能够解码
			if _, decodeErr := DecodeErrorID(err.ID); decodeErr != nil {
				t.Errorf("%s 错误ID解码失败: %v", tc.name, decodeErr)
			}
		})
	}
}

func TestErrorIDWithMetadata(t *testing.T) {
	// 测试带有元数据的错误ID处理
	err := New(500, "DB_ERROR", "数据库错误").
		WithMetadata(map[string]string{
			"table":     "users",
			"operation": "select",
		})

	if err.ID == "" {
		t.Error("带元数据的错误应该有错误ID")
	}

	// 验证WithMetadata不会影响错误ID
	originalID := err.ID
	errWithMoreMeta := err.WithMetadata(map[string]string{
		"table":     "users",
		"operation": "select",
		"query":     "SELECT * FROM users WHERE id = ?",
	})

	if errWithMoreMeta.ID != originalID {
		t.Error("WithMetadata不应该改变错误ID")
	}
}

func TestErrorIDFromError(t *testing.T) {
	// 测试FromError函数处理错误ID
	originalErr := New(400, "ORIG", "原始错误")
	originalID := originalErr.ID

	// 通过FromError转换
	convertedErr := FromError(originalErr)

	if convertedErr.ID != originalID {
		t.Error("FromError应该保持原有的错误ID")
	}

	// 测试从标准错误转换
	stdErr := &Error{Status: Status{Code: 500, Reason: "STD", Message: "标准错误"}}
	convertedStdErr := FromError(stdErr)

	if convertedStdErr.ID == "" {
		t.Error("FromError转换标准错误时应该生成错误ID")
	}
}

func TestErrorIDBase64Encoding(t *testing.T) {
	// 测试错误ID确实是base64编码的
	err := New(200, "OK", "成功")

	// 尝试解码base64
	decoded, decodeErr := base64.StdEncoding.DecodeString(err.ID)
	if decodeErr != nil {
		t.Errorf("错误ID应该是有效的base64编码: %v", decodeErr)
	}

	decodedStr := string(decoded)
	if decodedStr == "" {
		t.Error("解码后的字符串不应该为空")
	}

	// 验证解码后的字符串包含预期的格式
	// 格式: pkg.func@file:line:timestamp:gid:pid:random
	parts := strings.Split(decodedStr, ":")
	if len(parts) < 6 {
		t.Errorf("解码后的信息格式不正确，期望至少6个部分，实际: %d, 内容: %s", len(parts), decodedStr)
	}

	// 验证包含必要的信息
	if !strings.Contains(decodedStr, "@") {
		t.Errorf("解码信息应该包含@分隔符，实际: %s", decodedStr)
	}

	if !strings.Contains(decodedStr, "errors_test.go") {
		t.Errorf("解码信息应该包含文件名，实际: %s", decodedStr)
	}
}

func TestWithID(t *testing.T) {
	// 测试自定义错误ID
	customID := "custom-error-id-12345"
	err := New(400, "CUSTOM", "自定义ID错误").WithID(customID)

	if err.ID != customID {
		t.Errorf("自定义错误ID设置失败，期望: %s，实际: %s", customID, err.ID)
	}

	// 测试GetID方法
	retrievedID := err.GetID()
	if retrievedID != customID {
		t.Errorf("GetID返回的ID不正确，期望: %s，实际: %s", customID, retrievedID)
	}
}

func TestGetIDGeneration(t *testing.T) {
	// 测试GetID在没有ID时自动生成
	err := &Error{
		Status: Status{
			Code:    404,
			Reason:  "NOT_FOUND",
			Message: "未找到",
			// ID留空
		},
	}

	generatedID := err.GetID()
	if generatedID == "" {
		t.Error("GetID应该在没有ID时自动生成一个")
	}

	if err.ID != generatedID {
		t.Error("GetID生成的ID应该被设置到错误对象中")
	}

	// 再次调用GetID应该返回相同的ID
	secondCall := err.GetID()
	if secondCall != generatedID {
		t.Error("多次调用GetID应该返回相同的ID")
	}
}

// Benchmark测试
func BenchmarkErrorIDGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(400, "BENCH", "基准测试错误")
	}
}

func BenchmarkErrorIDDecoding(b *testing.B) {
	err := New(400, "BENCH", "基准测试错误")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeErrorID(err.ID)
	}
}
