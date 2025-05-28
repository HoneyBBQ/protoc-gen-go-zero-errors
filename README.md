# protoc-gen-go-zero-errors

一个为 go-zero 项目提供 go-kratos 风格错误处理的 protoc 插件。

## ✨ 特性

- 🔥 **仿照 go-kratos 风格的错误处理**
- 🚀 **专为 go-zero 项目设计**  
- 📝 **从 protobuf 生成错误代码**
- 🛠️ **支持自定义 HTTP 状态码**
- 📖 **从 proto 注释自动生成文档**
- 🔗 **完整的错误链支持**
- 🌐 **支持 Buf Schema Registry**
- 🔍 **智能错误ID追踪** - 包含文件名、函数名、行号、时间戳、Goroutine ID等debug信息

## 🚀 快速开始

### 使用 Buf Remote Plugin（推荐）

如果插件已发布到 Buf Schema Registry，您可以直接使用远程插件：

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/honeybbq/protoc-gen-go-zero-errors:v1.0.0
    out: gen/go
    opt:
      - paths=source_relative
```

```bash
buf generate
```

### 本地安装

```bash
go install github.com/honeybbq/protoc-gen-go-zero-errors@latest
```

或者从源码构建：

```bash
git clone https://github.com/honeybbq/protoc-gen-go-zero-errors.git
cd protoc-gen-go-zero-errors
go build -o protoc-gen-go-zero-errors .
# 将可执行文件移动到 PATH 中
mv protoc-gen-go-zero-errors $GOPATH/bin/
```

### 基本用法

1. **定义错误 proto 文件**

```protobuf
syntax = "proto3";

package api.user.v1;

import "proto/errors/options.proto";

option go_package = "github.com/your-project/api/user/v1;userv1";

enum UserError {
  option (errors.default_code) = 500;

  // 用户未找到
  USER_NOT_FOUND = 0 [(errors.code) = 404];
  // 用户已存在  
  USER_ALREADY_EXISTS = 1 [(errors.code) = 409];
  // 密码错误
  INVALID_PASSWORD = 2 [(errors.code) = 401];
}
```

2. **生成错误代码**

```bash
# 使用 protoc
protoc --proto_path=. --go_out=. --go_opt=paths=source_relative \
       --go-zero-errors_out=. --go-zero-errors_opt=paths=source_relative \
       api/user/v1/errors.proto

# 或使用 buf
buf generate
```

3. **在 go-zero 中使用**

```go
// logic 层
func (l *GetUserLogic) GetUser(req *types.GetUserReq) (*types.GetUserResp, error) {
    user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.Id)
    if err != nil {
        return nil, userv1.ErrorUserNotFound("用户ID %d 不存在", req.Id)
    }
    return &types.GetUserResp{Id: user.Id, Name: user.Name}, nil
}

// 错误检查
if userv1.IsUserNotFound(err) {
    // 处理用户未找到错误
}

// 获取错误ID用于日志记录和追踪
errorID := errors.ID(err)
log.Printf("错误发生 [ID: %s]: %v", errorID, err)
```

## 🔍 错误ID追踪功能

### 自动生成的错误ID包含：
- 📁 **包名** - 错误发生的包
- 🔧 **函数名** - 具体的函数位置
- 📄 **文件名** - 源代码文件
- 📍 **行号** - 精确的代码位置
- ⏰ **纳秒时间戳** - 错误发生的精确时间
- 🧵 **Goroutine ID** - 并发环境中的协程标识
- 🆔 **进程ID** - 多进程环境中的进程标识
- 🎲 **随机后缀** - 避免时间戳冲突

### 使用示例：

```go
// 创建错误
err := errors.NotFound("USER_NOT_FOUND", "用户不存在")

// 获取错误ID
errorID := err.GetID() // 或 errors.ID(err)
fmt.Printf("错误ID: %s", errorID)

// 解码错误ID (仅开发环境)
if debugInfo, err := errors.DecodeErrorID(errorID); err == nil {
    fmt.Printf("Debug信息: %s", debugInfo["raw"])
    // 输出类似: api/user/v1.GetUser@user_logic.go:25:1640995200123456789:1:12345:a1b2c3d4
}

// HTTP响应中自动包含错误ID
{
  "code": 404,
  "reason": "USER_NOT_FOUND", 
  "message": "用户不存在",
  "id": "YXBpL3VzZXIvdjEuR2V0VXNlckB1c2VyX2xvZ2ljLmdvOjI1OjE2NDA5OTUyMDA="
}
```

## 📦 项目结构

```
protoc-gen-go-zero-errors/
├── main.go                    # protoc插件主程序
├── errors.go                  # 代码生成逻辑
├── errors/                    # 错误处理核心包
│   └── errors.go              # go-kratos风格的错误处理库
├── interceptor/               # 拦截器
│   ├── http.go                # HTTP拦截器 (支持错误ID)
│   └── grpc.go                # gRPC拦截器 (支持错误ID)
└── proto/                     # protobuf扩展定义
    └── errors/
        ├── options.proto      # 错误码扩展选项
        └── errors.proto       # 错误状态定义

```

## 🎯 核心 API

### 错误创建

- `New(code, reason, message)` - 创建新错误 (自动生成ID)
- `Newf(code, reason, format, args...)` - 创建格式化错误
- `BadRequest()`, `Unauthorized()`, `Forbidden()`, `NotFound()` 等便利函数

### 错误检查  

- `Code(err)` - 获取错误代码
- `Reason(err)` - 获取错误原因
- `ID(err)` - 获取错误ID (新增)
- `IsBadRequest()`, `IsNotFound()` 等检查函数

### 错误管理

- `FromError(err)` - 从任意错误转换
- `GRPCStatus()` - 转换为 gRPC 状态 (包含错误ID)
- `WithID(id)` - 设置自定义错误ID
- `DecodeErrorID(id)` - 解码错误ID获取debug信息

### 错误转换

- `ToHTTPCode()` / `ToGRPCCode()` - 状态码转换

## 🔧 拦截器集成

### HTTP拦截器

```go
import "github.com/honeybbq/protoc-gen-go-zero-errors/interceptor"

// 设置默认错误处理器 (自动包含错误ID)
interceptor.SetDefaultErrorHandler()

// 或使用中间件
app.Use(interceptor.HTTPErrorMiddleware)
```

### gRPC拦截器

```go
import "github.com/honeybbq/protoc-gen-go-zero-errors/interceptor"

// Unary拦截器
s := grpc.NewServer(
    grpc.UnaryInterceptor(interceptor.UnaryServerErrorInterceptor()),
    grpc.StreamInterceptor(interceptor.StreamServerErrorInterceptor()),
)
```

## 🔧 Buf 配置

创建 `buf.gen.yaml` 文件：

```yaml
version: v1
plugins:
  # 生成标准 go protobuf 代码
  - plugin: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative
  
  # 生成 go-zero 风格错误代码  
  - plugin: go-zero-errors
    out: gen/go
    opt:
      - paths=source_relative 
```

## 📝 与 go-kratos 的关系

本项目参考了 go-kratos/errors 的设计理念，但专为 go-zero 优化：

| 功能特性 | bytenet-errors-gen | 说明 |
|---------|-------------------|------|
| `New()` | ✅ 支持 | 创建错误 |
| `Code()` | ✅ 支持 | 获取错误码 |
| `Reason()` | ✅ 支持 | 获取错误原因 |
| `BadRequest()` | ✅ 支持 | HTTP 400 错误 |
| `NotFound()` | ✅ 支持 | HTTP 404 错误 |
| `FromError()` | ✅ 支持 | 错误转换 |
| protobuf 扩展 | ✅ 参考实现 | 1109, 1108 |

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT License 