# benzhi.Dockerfile - 轻量级延迟任务调度器
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 启用 CGO 以保证 -race 竞态检测可用
ENV CGO_ENABLED=1

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 预编译验证
RUN go build ./...

# 默认启动命令
CMD ["go", "run", "./cmd/server"]