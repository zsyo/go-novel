#!/bin/bash

# 设置项目名称
PROJECT_NAME="go-novel"

# 设置目标平台
GOOS=linux
GOARCH=amd64

echo "开始构建 $GOOS/$GOARCH 版本的 $PROJECT_NAME..."

# 构建二进制文件（去除调试信息的最小化编译）
echo "正在编译..."
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -a -trimpath -ldflags="-s -w" -o $PROJECT_NAME .

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo "编译失败，停止构建过程"
    exit 1
fi

# 检查UPX是否可用
if command -v upx &> /dev/null
then
    echo "正在使用UPX压缩二进制文件..."
    upx -9 $PROJECT_NAME
    
    # 检查压缩是否成功
    if [ $? -ne 0 ]; then
        echo "UPX压缩失败，但继续构建过程"
    fi
else
    echo "警告: UPX未安装，跳过压缩步骤"
fi

# 构建Docker镜像
echo "正在构建Docker镜像..."
docker build -t $PROJECT_NAME:latest .

# 检查Docker构建是否成功
if [ $? -ne 0 ]; then
    echo "Docker镜像构建失败，停止构建过程"
    exit 1
fi

# 清理
echo "清理临时文件..."
rm -f ./$PROJECT_NAME

echo "构建完成！"
echo ""
echo "使用以下命令运行容器："
echo "docker run -d --user=\$(id -u):\$(id -g) -p 7765:7765 --name go-novel go-novel:latest"
echo ""
echo "或者挂载本地配置目录："
echo "docker run -d \\"
echo "  --user=\$(id -u):\$(id -g) \\"
echo "  -p 7765:7765 \\"
echo "  -v \$(pwd)/configs:/app/configs \\"
echo "  --name go-novel \\"
echo "  go-novel:latest"
echo ""
echo "或者挂载本地下载目录："
echo "docker run -d \\"
echo "  --user=\$(id -u):\$(id -g) \\"
echo "  -p 7765:7765 \\"
echo "  -v \$(pwd)/downloads:/app/downloads \\"
echo "  --name go-novel \\"
echo "  go-novel:latest"
echo ""
echo "或者同时挂载配置和下载目录："
echo "docker run -d \\"
echo "  --user=\$(id -u):\$(id -g) \\"
echo "  -p 7765:7765 \\"
echo "  -v \$(pwd)/configs:/app/configs \\"
echo "  -v \$(pwd)/downloads:/app/downloads \\"
echo "  --name go-novel \\"
echo "  go-novel:latest"
echo ""
echo "同时挂载配置和下载目录（精细挂载配置、规则目录）: "
echo "docker run -d \\"
echo "  --user=\$(id -u):\$(id -g) \\"
echo "  -p 7765:7765 \\"
echo "  -v \$(pwd)/configs/config.ini:/app/configs/config.ini \\"
echo "  -v \$(pwd)/configs/rules:/app/configs/rules \\"
echo "  -v \$(pwd)/downloads:/app/downloads \\"
echo "  --name go-novel \\"
echo "  go-novel:latest"