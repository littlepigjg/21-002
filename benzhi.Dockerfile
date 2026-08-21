# benzhi.Dockerfile - Go 项目评测镜像
# 注意：容器内必须保留完整 Go 工具链，不能使用多阶段编译

FROM golang:1.22

WORKDIR /app

# 复制所有源代码（本项目仅用标准库，无需 go mod download）
COPY . .

# 注意：不做构建时预编译，跨架构 QEMU 下 Go/GCC 偶发 segfault
# 编译验证统一放到容器运行时（步骤 3）执行。

# 默认启动命令
CMD ["go", "run", "./cmd/server"]
