# 构建阶段
FROM golang:1.26 AS builder

WORKDIR /app

# 复制 go.work 和 modbus 目录
COPY go.work .
COPY modbus/ ./modbus/

# 设置 Go 代理
# ENV GOPROXY=https://proxy.golang.org,direct
# ENV GOSUMDB=off

# 进入 modbus 目录构建
WORKDIR /app/modbus
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/sbcc ./main/run.go

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装 CA 证书和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 复制可执行文件
COPY --from=builder /app/sbcc /app/sbcc

# 暴露端口
EXPOSE 9081

# 设置时区
ENV TZ=Asia/Shanghai

# 启动命令
CMD ["/app/sbcc"]
