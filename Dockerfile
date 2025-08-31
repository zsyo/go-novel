# 使用最小化的Alpine Linux作为基础镜像
FROM alpine:latest

# 设置镜像源为阿里云镜像站（可选，提高下载速度）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装ca-certificates以支持HTTPS请求
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN adduser -D -s /bin/sh go-novel

# 设置工作目录
WORKDIR /app

# 复制二进制文件
COPY go-novel ./

# 更改文件所有者
RUN chown -R go-novel:go-novel /app

# 切换到非root用户
USER go-novel

# 暴露端口
EXPOSE 7765

# 设置入口点
ENTRYPOINT ["./go-novel"]