FROM golang:1.27.4-alpine AS builder

# 设置环境变量
ENV GOPROXY=https://goproxy.cn,direct

# 设置工作目录（构建上下文为仓库根，go.work 在根）
WORKDIR /build

# 复制整个项目代码（.dockerignore 已排除 node_modules/log/output/storage/.git 等）
COPY . .

# 下载依赖（go.work 下两个模块：./apps/api 与 ./pkg）
RUN go mod download

# 构建应用程序（入口文件在 apps/api/cmd，模块 github.com/morehao/ark-go-starter/apps/api）
RUN go build -ldflags="-s -w" -o /app/demo ./apps/api/cmd

# 运行阶段
FROM alpine:3.22

# 运行阶段安装时区库：config.prod.yaml 的 PG DSN 带 TimeZone=Asia/Shanghai，
# alpine 基础镜像不含 tzdata，缺了会 "unknown time zone Asia/Shanghai" 导致打开 DB 失败、容器 panic。
RUN apk add --no-cache tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /app/demo /app/demo

# 从构建阶段复制生产环境配置文件
COPY --from=builder /build/apps/api/config/config.prod.yaml /app/config.yaml

# 设置工作目录
WORKDIR /app

# 暴露端口
EXPOSE 8099

# 启动应用程序
ENTRYPOINT ["/app/demo"]
