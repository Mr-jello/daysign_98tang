# daysign_98tang

本项目是一个在98tang自动回帖与签到程序，通过 [chromedp](https://github.com/chromedp/chromedp) 实现网页自动化操作，同时使用 [Telegram Bot API](https://core.telegram.org/bots/api) 在签到成功后发送通知。

## 特性

- 使用 chromedp 实现网页自动化操作，使用 headless 模式，模拟人工操作
- 自动登录、保存并加载 cookies
- 随机等待时间，避免被检测(定时任务的时间 + 自定义随机等待时间s)
- 自动回帖与签到操作
- 签到成功后发送 Telegram 通知
- 内置 Makefile 支持跨平台构建
- 使用 GitHub Action 自动构建发布

## 前置条件

- chrome/chromium
- Go 1.24+

## 使用说明

### 1. 本地构建与测试

**修改 `.env.example` 文件为 `.env`，并填入你的配置信息**

```bash
go run main.go

# 或者使用 Makefile 构建
make build/make build-all
./build/daysign98tang_mac
```

### 2. 服务器部署

1. 安装chrome/chromium

```bash
# Ubuntu
sudo apt-get update
sudo apt-get install -y chromium-browser
```


2. 克隆本项目到服务器

```bash
git clone https://github.com/Mr-jello/daysign_98tang.git
cd daysign_98tang
```

3. 复制环境变量模板文件：

```bash
cp .env.example .env
```
- 修改 .env 文件，填入你的配置信息

4. 执行 Makefile 构建

```bash
make build-all
```
- 选择适合服务器架构，将生成的二进制文件 `daysign98tang_xx` 修改为 `daysign98tang`上传到服务器
- 例如：/root目录下

5. 执行二进制文件（有日志文件，所以不需要重定向输出）

```bash
nohup ./daysign98tang &
```

### 3. 使用 Docker 部署

```bash
docker build -t daysign98tang .
```

## 未来计划
- [x] 使用环境变量配置信息用户信息, 系统配置信息
- [x] 通知信息更加详细
- [ ] 支持更多通知方式，如钉钉、企业微信，邮箱等

