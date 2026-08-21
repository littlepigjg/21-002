# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

# 禁用 cgo，使用纯 Go 编译（项目仅依赖标准库，避免 QEMU 跨架构模拟时 gcc segfault）
ENV CGO_ENABLED=0

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]