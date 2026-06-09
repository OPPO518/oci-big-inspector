# ==========================================
# 阶段一：在 Node.js 环境中构建 Vue 3 前端
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend

# 复制前端配置文件并安装依赖
COPY ./frontend/package*.json ./
RUN npm install

# 复制所有前端源码并打包出 dist 目录
COPY ./frontend .
RUN npm run build

# ==========================================
# 阶段二：在 Go 环境中将前端资产静态嵌入并编译
# ==========================================
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app

# 【修复点 1】把 backend 目录下所有的 .go 和 .mod 文件全部复制进来
COPY ./backend ./

# 【修复点 2】让 Go 自动下载 SQLite 驱动和甲骨文 SDK 等所有依赖
RUN go mod tidy

# 从前端阶段复制编译好的 dist 目录到 Go 的上下文中
COPY --from=frontend-builder /frontend/dist ./dist

# 【修复点 3】将结尾的 main.go 改成 . ，表示把当前目录下的所有 Go 文件联合编译
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o big-inspector .

# ==========================================
# 阶段三：最终极简运行环境（只有这一个容器投产）
# ==========================================
FROM alpine:3.19
# 安装 acme.sh 运行必备的系统工具
RUN apk add --no-cache ca-certificates curl openssl socat bash

WORKDIR /app

# 把最终的单文件二进制偷过来
COPY --from=backend-builder /app/big-inspector .

# 在容器内全局安装 acme.sh，并强行将证书锁在挂载目录
ENV LE_CONFIG_HOME=/app/data/.acme.sh
RUN curl https://get.acme.sh | sh

# 暴露 80 和 443 端口
EXPOSE 80
EXPOSE 443

CMD ["./big-inspector"]
