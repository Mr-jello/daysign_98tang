#!/bin/bash

# 终止所有Chrome进程
pkill -9 -f chrom || true

# 等待进程完全终止
sleep 3

# 设置环境变量
export CHROME_PATH=/snap/bin/chromium
export ENABLE_HEADLESS=true
export FORCE_KILL_CHROME=true

# 在大多数无头服务器上，这些错误可以忽略
echo "启动程序..."

# 运行程序
nohup ./daysign98tang &
echo $! > daysign.pid
echo "程序已在后台启动，PID: $(cat daysign.pid)"
