# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译
# 使用 TARGETARCH 选择对应架构的 golang 基础镜像，避免 QEMU 下的编译问题

ARG TARGETARCH=amd64
FROM --platform=linux/${TARGETARCH} golang:1.22

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]