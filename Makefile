.PHONY: build test clean install error-decoder demo help

# 默认目标
all: build

# 构建主插件
build:
	@echo "🔨 构建 protoc-gen-go-zero-errors..."
	go build -o protoc-gen-go-zero-errors .

# 构建错误ID解析工具
error-decoder:
	@echo "🔧 构建错误ID解析工具..."
	cd cmd/error-decoder && go build -o ../../error-decoder .

# 构建所有工具
build-all: build error-decoder
	@echo "✅ 所有工具构建完成！"

# 运行测试
test:
	@echo "🧪 运行测试..."
	cd errors && go test -v
	@echo "📊 运行基准测试..."
	cd errors && go test -bench=. -benchmem

# 测试错误ID功能
test-error-id:
	@echo "🔍 测试错误ID功能..."
	cd errors && go test -v -run TestErrorID

# 安装插件到GOPATH
install:
	@echo "📦 安装插件到 GOPATH..."
	go install .

# 演示错误ID解析工具
demo: error-decoder
	@echo "🎭 演示错误ID解析工具..."
	@echo "请先运行测试生成错误ID，然后手动测试解析工具："
	@echo "./error-decoder -h"

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -f protoc-gen-go-zero-errors
	rm -f error-decoder

# 显示帮助信息
help:
	@echo "🚀 Go-Zero 错误处理插件 - Makefile 帮助"
	@echo "================================================"
	@echo ""
	@echo "构建命令:"
	@echo "  make build         - 构建主插件"
	@echo "  make error-decoder - 构建错误ID解析工具"
	@echo "  make build-all     - 构建所有工具"
	@echo "  make install       - 安装插件到 GOPATH"
	@echo ""
	@echo "测试命令:"
	@echo "  make test          - 运行所有测试"
	@echo "  make test-error-id - 仅测试错误ID功能"
	@echo ""
	@echo "演示命令:"
	@echo "  make demo          - 演示错误ID解析工具"
	@echo ""
	@echo "其他命令:"
	@echo "  make clean         - 清理构建文件"
	@echo "  make help          - 显示此帮助信息"

# 构建Docker镜像
docker-build:
	@echo "🐳 构建Docker镜像..."
	docker build -t protoc-gen-go-zero-errors:latest .

# 运行Docker容器测试
docker-test: docker-build
	@echo "🧪 在Docker中测试..."
	docker run --rm protoc-gen-go-zero-errors:latest --version 