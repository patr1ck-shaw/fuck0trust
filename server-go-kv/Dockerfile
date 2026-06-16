# 构建阶段
FROM golang:1.22-alpine AS builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY main.go ./

# 构建二进制文件（静态链接）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fuck0trust-server .

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装 ca-certificates（HTTPS 需要）
RUN apk --no-cache add ca-certificates tzdata && \
    mkdir -p /app/data && \
    adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/fuck0trust-server /app/

USER appuser

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

CMD ["/app/fuck0trust-server"]
