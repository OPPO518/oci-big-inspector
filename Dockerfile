# ==========================================
# 阶段一：在 Node.js 环境中构建 Vue 3 前端
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
# 【占位逻辑】在后续实际开发前端前，先生成一个干净的 dist 目录防止 Go 编译报错
RUN mkdir dist && echo "<h1>大探长 3x-ui 增强版控制台</h1><p>全容器化闭环、支持纯 IP 自动证书。</p>" > dist/index.html

# ==========================================
# 阶段二：在 Go 环境中将前端资产静态嵌入并编译
# ==========================================
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
COPY ./backend/go.mod ./
# 如果后续引入了第三方包，在这里取消注释
# RUN go mod download

COPY ./backend/main.go .
COPY --from=frontend-builder /frontend/dist ./dist

# 纯静态编译，彻底剥离 CGO 依赖，极致压榨体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o big-inspector main.go

# ==========================================
# 阶段三：最终产线运行环境（单容器完全体）
# ==========================================
FROM alpine:3.19
# 安装 acme.sh 运行和验证所需的系统组件
RUN apk add --no-cache ca-certificates curl openssl socat bash

WORKDIR /app

# 从第二阶段偷走最终的单文件二进制程序
COPY --from=backend-builder /app/big-inspector .

# 在容器内全局安装 acme.sh，并将证书生成路径强行锁在我们的数据挂载目录下
ENV LE_CONFIG_HOME=/app/data/.acme.sh
RUN curl https://get.acme.sh | sh

# 暴露 80（ACME Standalone 强占用）和用户控制台端口
EXPOSE 80
EXPOSE 443

CMD ["./big-inspector"]
