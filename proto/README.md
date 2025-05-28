# go-zero-errors Proto Extensions

这个模块提供了用于 `protoc-gen-go-zero-errors` 插件的 protobuf 扩展定义。

## 功能

- `(errors.code)` - 为枚举值指定 HTTP 状态码
- `(errors.default_code)` - 为枚举指定默认 HTTP 状态码

## 发布到 BSR

这个模块已发布到 Buf Schema Registry，可以通过以下方式引用：

```yaml
# buf.yaml
version: v2
deps:
  - buf.build/bbq/go-zero-errors
```

## 使用方法

在你的 proto 文件中：

```protobuf
syntax = "proto3";

import "errors/options.proto";

enum MyError {
  option (errors.default_code) = 500;
  
  NOT_FOUND = 0 [(errors.code) = 404];
  ALREADY_EXISTS = 1 [(errors.code) = 409];
}
```

## 字段编号

- `code`: 1109
- `default_code`: 1108