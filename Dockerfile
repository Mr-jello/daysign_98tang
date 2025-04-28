# syntax=docker/dockerfile:1

#######################
# 构建阶段 (builder)
#######################
FROM golang:1.24-alpine AS builder
WORKDIR /daysign

# 安装必要的构建工具
RUN apk add --no-cache make git

# 设置 GOPROXY 为国内镜像（根据需要可修改）
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 设置编译环境变量，禁用 CGO
ENV CGO_ENABLED=0

# 编译目标二进制文件（GOOS 和 GOARCH 会在 buildx 构建时指定）
RUN go build -ldflags="-s -w" -o daysign98tang main.go

#######################
# 运行阶段 (final)
#######################
FROM alpine:latest

# 安装 Chromium 和相关依赖（用于 headless 模式）
RUN apk add --no-cache \
    chromium \
    chromium-chromedriver \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    font-noto \
    tzdata \
    ca-certificates

# 设置时区为亚洲/上海
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /daysign/daysign98tang /app/

# 创建必要的目录结构
RUN mkdir -p /app/logs

# 声明卷挂载点
VOLUME ["/app/logs", "/app/cookies", "/app/.env"]

# 容器启动命令
CMD ["/app/daysign98tang"]