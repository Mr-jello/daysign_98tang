#!/bin/bash

# 检查 Chrome 可执行文件
if [ -f /snap/bin/chromium ]; then
    CHROME_EXECUTABLE="/snap/bin/chromium"
elif [ -f /usr/bin/chromium-browser ]; then
    CHROME_EXECUTABLE="/usr/bin/chromium-browser"
elif [ -f /usr/bin/google-chrome ]; then
    CHROME_EXECUTABLE="/usr/bin/google-chrome"
else
    echo "未找到 Chrome/Chromium 可执行文件，尝试安装..."
    apt-get update && apt-get install -y chromium
    CHROME_EXECUTABLE="/snap/bin/chromium"
fi

# 终止所有Chrome进程
pkill -9 -f chrom || true
echo "已终止所有Chrome进程"

# 等待进程完全终止
sleep 3

# 设置环境变量
export CHROME_PATH="$CHROME_EXECUTABLE"
export ENABLE_HEADLESS=true
export FORCE_KILL_CHROME=true
export NO_SANDBOX=true  # 通知程序使用--no-sandbox

# 追加Chrome配置到.env文件
if [ -f .env ]; then
    echo "找到现有的.env文件，追加Chrome配置..."
    
    # 检查文件中是否已存在相关配置，如果存在则更新
    if grep -q "CHROME_PATH=" .env; then
        sed -i "s|CHROME_PATH=.*|CHROME_PATH=$CHROME_EXECUTABLE|" .env
    else
        echo "" >> .env  # 添加一个空行
        echo "# Chrome配置" >> .env
        echo "CHROME_PATH=$CHROME_EXECUTABLE" >> .env
    fi
    
    if ! grep -q "ENABLE_HEADLESS=" .env; then
        echo "ENABLE_HEADLESS=true" >> .env
    fi
    
    if ! grep -q "NO_SANDBOX=" .env; then
        echo "NO_SANDBOX=true" >> .env
    fi
    
    echo "已更新.env文件中的Chrome配置"
else
    echo ".env文件不存在，请使用模版创建一个"
fi

# 在大多数无头服务器上，DBus和AppArmor错误可以忽略
echo "启动程序..."
echo "使用Chrome路径: $CHROME_PATH"

# 停止之前可能运行的实例
if [ -f daysign98tang.pid ]; then
    OLD_PID=$(cat daysign98tang.pid)
    if ps -p $OLD_PID > /dev/null; then
        echo "终止旧进程 PID: $OLD_PID"
        kill -9 $OLD_PID || true
    fi
fi

# 用正确的环境变量启动程序
CHROME_PATH="$CHROME_EXECUTABLE" NO_SANDBOX=true nohup ./daysign98tang &

# 将进程ID保存到文件
echo $! > daysign98tang.pid
echo "程序已在后台启动，PID: $(cat daysign98tang.pid)"
echo "当前配置:"
echo "Chrome路径: $CHROME_PATH"

# 删除nohup.out文件
if [ -f nohup.out ]; then
    rm nohup.out 
fi