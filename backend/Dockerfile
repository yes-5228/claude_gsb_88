# ---------- 构建阶段：编译 Go 静态二进制 ----------
FROM golang:1.27-alpine AS build
WORKDIR /src

# CGO 关闭后可以编译出静态二进制；GOPROXY 走国内镜像，容器内拉依赖更稳定
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

# 先复制依赖清单预热模块缓存，源码改动时可以复用该层
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags "-s -w" -o /out/server ./cmd/server

# ---------- 运行阶段：仅保留二进制与运行时依赖 ----------
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai \
    DB_DRIVER=postgres \
    DB_HOST=postgres \
    DB_PORT=5432 \
    DB_NAME=desilting \
    DB_USER=desilting \
    DB_SSLMODE=disable \
    DB_TIMEZONE=Asia/Shanghai \
    APP_PORT=8080 \
    APP_ENV=production \
    APP_SEED_ENABLED=true

COPY --from=build /out/server /app/server

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=10 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/server"]
