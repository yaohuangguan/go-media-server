# ---------------------------------------------------
# 第一阶段：编译 Go 程序 (Builder)
# ---------------------------------------------------
    FROM golang:1.23-alpine AS builder

    # 设置工作目录
    WORKDIR /app
    
    # 复制依赖文件并下载
    COPY go.mod go.sum ./
    # 设置国内代理，虽然 Cloud Run 构建在海外，但为了保险
    ENV GOPROXY=https://goproxy.cn,direct
    RUN go mod download
    
    # 复制源码
    COPY . .
    
    # 编译 Go 程序
    # CGO_ENABLED=0 表示静态编译，不依赖系统库，兼容性最好
    RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go
    
    # ---------------------------------------------------
    # 第二阶段：构建运行环境 (Runner)
    # ---------------------------------------------------
    # 使用 Alpine Linux，因为它体积极小且安全
    FROM alpine:latest
    
    # 1. 安装基础依赖：Python3, FFmpeg, Curl, CA证书
    RUN apk add --no-cache \
        python3 \
        ffmpeg \
        curl \
        ca-certificates
    
    # 2. 安装 yt-dlp (下载 Linux 二进制文件)
    # 直接从 GitHub 下载最新版到 /usr/local/bin 并赋予执行权限
    RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
        chmod a+rx /usr/local/bin/yt-dlp
    
    # 3. 从第一阶段复制编译好的 Go 程序
    WORKDIR /app
    COPY --from=builder /app/server .
    
    # 4. 告诉 Cloud Run 我们的服务监听哪个端口 (虽然 Cloud Run 会覆盖这个)
    ENV PORT=8080
    
    # 5. 启动服务
    CMD ["./server"]