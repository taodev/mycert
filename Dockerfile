# ===== 构建阶段 =====
FROM golang:1.24.3-alpine3.21 AS builder

# 安装必要依赖（如有 C 库或 git 等依赖）
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /build

# 复制源码
COPY . ./

# 拉取依赖并构建
RUN go mod download
RUN go build -o mycert

# ===== 运行阶段 =====
FROM alpine:3.21

# 安装 CA 证书，避免 HTTPS 请求失败（可选）
RUN apk add --no-cache ca-certificates

# 创建工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /build/mycert /app/mycert

# 复制 mkcert
COPY ./mkcert /app/mkcert

# 授权执行权限（可选）
RUN chmod +x /app/mycert && chmod +x /app/mkcert && mkdir -p /mycert

ENTRYPOINT ["/app/mycert"]
