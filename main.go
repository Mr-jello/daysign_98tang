package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

// 论坛选择器
const (
	// 确认年龄
	ConfirmAgeSelector = `.domain`
	// 登陆按钮
	EnterButtonSelector = `.enter-btn`
	// Header 选择器
	HeaderSelector = `#hd`
	// 登陆表单
	LoginFormSelector = `.y.pns`
	// 登陆用户名输入框
	UsernameSelector = `#ls_username`
	// 登陆密码输入框
	PasswordSelector = `#ls_password`
	// 登陆按钮
	LoginButtonSelector = `.pn.vm`
	// 二步认证表单
	TwoFactorSelector = `#fwin_login`
	// 二步认证安全问题答案输入框
	TwoFactorAnswerSelector = `.px.p_fre`
	// 二步认证按钮
	TwoFactorButtonSelector = `.pn.pnc`
	// 综合讨论区主要内容
	ReplyContentsSelector = `#threadlisttableid`
	// 用户普通帖子
	ReplyNormalThread = `#normalthread_`
	// 用户回帖输入框
	ReplyInputSelector = `#fastpostmessage`
	// 用户回帖按钮
	ReplyButtonSelector = `#fastpostsubmit`
	// 论坛签到按钮
	CheckInButtonSelector = `.ddpc_sign_btn_red`
	// 签到验证
	CheckInVerifyFormSelector = `#fwin_pc_click_ddsign`
	// 签到确认按钮
	CheckInVerifyButtonSelector = `.pn.pnc`
	// 已签到区域
	CheckInDoneSelector = `.ddpc_sign_btna`
	// 金钱区域
	MyCoinsSelector = `.xi1.cl`
)

// 回帖的内容
var ReplyContents = []string{
	"感谢楼主分享好片",
	"感谢分享！！",
	"谢谢分享！",
	"感谢分享感谢分享",
	"必需支持",
	"简直太爽了",
	"感谢分享啊",
	"封面还不错",
	"有点意思啊",
	"封面还不错，支持一波",
	"真不错啊",
	"不错不错",
	"这身材可以呀",
	"终于等到你",
	"謝謝辛苦分享",
	"赏心悦目",
	"快乐无限~~",
	"這怎麼受的了啊",
	"谁也挡不住！",
	"分享支持。",
	"这谁顶得住啊",
	"这是要精J人亡啊!",
	"饰演很赞",
	"這系列真有戲",
	"感谢大佬分享v",
	"看着不错",
	"感谢老板分享",
	"可以看看",
	"谢谢分享！！！",
	"真是骚气十足",
	"给我看硬了！",
	"这个眼神谁顶得住。",
	"妙不可言",
	"看硬了，确实不错。",
	"这个我是真的喜欢",
	"如何做到像楼主一样呢",
	"分享一下技巧楼主",
	"身材真不错啊",
	"真是极品啊",
	"这个眼神谁顶得住。",
	"妙不可言",
	"感谢分享这一部资源",
	"终于来了，等了好久了。",
	"等这一部等了好久了！",
	"确实不错。",
	"真是太好看了",
}

// 全局变量，用于存储日志文件
var currentLogFile *os.File

// 全局任务状态和调度器
var (
	taskMutex       sync.Mutex
	isTaskRunning   bool
	lastRunTime     time.Time
	lastSuccessTime time.Time
	scheduler       *cron.Cron
	retryTimer      *time.Timer
)

// 环境变量
var (
	BaseURL          string
	ReplySection     string
	CheckInSection   string
	CoinsInfoSection string
	FormUsername     string
	FormPassword     string
	SecurityQuestion string
	SecurityAnswer   string
	ChatID           int64
	MyBotToken       string
	EnableHeadless   bool
	WaitingTime      int
	RetryInterval    time.Duration
	CronSchedule     string
	RunOnStart       bool
)

// Browser 结构体封装了 chromedp 的执行上下文，用于后续多步操作
type Browser struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Execute 用于执行一组 chromedp.Action，并设置一个超时
func (b *Browser) Execute(actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

// NavigateTo 导航到指定页面
func (b *Browser) NavigateTo(url string) error {
	return b.Execute(chromedp.Navigate(url))
}

// WaitForElement 等待页面中指定的元素可见
func (b *Browser) WaitForElement(selector string) error {
	return b.Execute(chromedp.WaitVisible(selector))
}

// ElementExists 检查页面中指定的元素是否存在
func (b *Browser) ElementExists(selector string) (bool, error) {
	var exists bool
	err := b.Execute(chromedp.Evaluate(`document.querySelector("`+selector+`") !== null`, &exists))
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetHTML 获取指定 js 路径对应的HTML内容
func (b *Browser) GetHTML(sel string) (string, error) {
	var html string
	err := b.Execute(chromedp.OuterHTML(sel, &html, chromedp.ByQuery))
	return html, err
}

// Click 模拟点击操作
func (b *Browser) Click(selector string) error {
	return b.Execute(chromedp.Click(selector, chromedp.ByQuery))
}

// Input 模拟输入文本
func (b *Browser) Input(selector, text string) error {
	return b.Execute(
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

// Close 关闭浏览器实例
func (b *Browser) Close() {
	b.cancel()
}

// init 函数
func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Fatalf("加载 .env 文件失败: %v", err)
	}

	// 初始化配置变量
	BaseURL = os.Getenv("BASE_URL")
	ReplySection = os.Getenv("REPLY_SECTION")
	CheckInSection = os.Getenv("CHECK_IN_SECTION")
	CoinsInfoSection = os.Getenv("COINS_INFO_SECTION")
	FormUsername = os.Getenv("FORUM_USERNAME")
	FormPassword = os.Getenv("FORUM_PASSWORD")
	SecurityQuestion = os.Getenv("SECURITY_QUESTION")
	SecurityAnswer = os.Getenv("SECURITY_ANSWER")

	// 转换 TELEGRAM_CHAT_ID 为 int64
	if chatIDStr := os.Getenv("TELEGRAM_CHAT_ID"); chatIDStr != "" {
		if id, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			ChatID = id
		}
	}

	MyBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	// 转化 ENABLE_HEADLESS 为 bool
	if enableHeadlessStr := os.Getenv("ENABLE_HEADLESS"); enableHeadlessStr != "" {
		if enable, err := strconv.ParseBool(enableHeadlessStr); err == nil {
			EnableHeadless = enable
		}
	}

	// 转化 WAITING_TIME 为 int
	if waitingTimeStr := os.Getenv("WAITING_TIME"); waitingTimeStr != "" {
		if waitingTime, err := strconv.Atoi(waitingTimeStr); err == nil {
			WaitingTime = waitingTime
		}
	}

	CronSchedule = os.Getenv("CRON_SCHEDULE")

	// 转化 RETRY_INTERVAL 为 duration
	if retryIntervalStr := os.Getenv("RETRY_INTERVAL"); retryIntervalStr != "" {
		if minutes, err := strconv.Atoi(retryIntervalStr); err == nil {
			RetryInterval = time.Duration(minutes) * time.Minute
		} else {
			// 尝试作为带单位的时间解析
			if duration, err := time.ParseDuration(retryIntervalStr); err == nil {
				RetryInterval = duration
			} else {
				log.Printf("无法解析重试间隔 '%s'，使用默认值30分钟", retryIntervalStr)
				RetryInterval = 30 * time.Minute
			}
		}
	} else {
		RetryInterval = 30 * time.Minute // 默认重试间隔为30分钟
	}

	// 转化 RUN_ON_START 为 bool
	if runOnStartStr := os.Getenv("RUN_ON_START"); runOnStartStr != "" {
		if runOnStart, err := strconv.ParseBool(runOnStartStr); err == nil {
			RunOnStart = runOnStart
		}
	}

	// 配置日志
	setupLogger()
}

// 设置日志
func setupLogger() {
	// 关闭之前的日志文件
	if currentLogFile != nil {
		currentLogFile.Close()
	}

	// 确保logs目录存在
	os.MkdirAll("logs", 0755)

	// 清理旧日志
	cleanupOldLogs(7)

	// 创建日志文件(日期为当天--当天+7天)
	logFileName := fmt.Sprintf("logs/98tang_daysign_%s.log", time.Now().Format("2006-01-02"))
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("无法创建日志文件: %v", err)
		return
	}

	// 同时输出到控制台和文件
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 保存当前日志文件指针
	currentLogFile = logFile
}

// 清理超过指定天数的旧日志
func cleanupOldLogs(daysToKeep int) {
	files, err := os.ReadDir("logs")
	if err != nil {
		log.Printf("读取日志目录失败: %v", err)
		return
	}

	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	// 日志文件名格式正则表达式
	logFilePattern := regexp.MustCompile(`98tang_daysign_(\d{4}-\d{2}-\d{2})\.log`)

	removed := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 匹配日志文件名
		matches := logFilePattern.FindStringSubmatch(file.Name())
		if len(matches) < 2 {
			continue
		}

		// 解析日志文件日期
		fileDate, err := time.Parse("2006-01-02", matches[1])
		if err != nil {
			log.Printf("无法解析日志文件日期 %s: %v", file.Name(), err)
			continue
		}

		// 如果文件日期早于截止日期，删除文件
		if fileDate.Before(cutoffDate) {
			if err := os.Remove(filepath.Join("logs", file.Name())); err != nil {
				log.Printf("删除过期日志文件 %s 失败: %v", file.Name(), err)
			} else {
				log.Printf("已删除过期日志文件: %s", file.Name())
				removed++
			}
		}
	}

	if removed > 0 {
		log.Printf("共清理了 %d 个过期日志文件", removed)
	}
}

// executeTask 执行完整的任务流程，任何步骤失败都会导致整个任务失败
func executeTask() {
	// 检查任务是否已经在运行
	taskMutex.Lock()
	if isTaskRunning {
		log.Println("任务已在运行中，跳过本次执行")
		taskMutex.Unlock()
		return
	}

	// 更新任务状态
	isTaskRunning = true
	lastRunTime = time.Now()
	taskMutex.Unlock()

	// 函数结束时清理状态
	defer func() {
		taskMutex.Lock()
		isTaskRunning = false
		taskMutex.Unlock()
	}()

	log.Println("开始执行任务...")

	// 收集任务结果
	var message strings.Builder
	currentTime := time.Now().Format("2006年01月02日 15:04:05")
	message.WriteString(fmt.Sprintf("%s 任务开始\n", currentTime))

	// 创建浏览器实例
	browser, err := NewBrowser()
	if err != nil {
		log.Printf("创建浏览器实例失败: %v", err)
		scheduleRetry("创建浏览器失败: " + err.Error())
		return
	}
	defer browser.Close()

	// 步骤1: 访问论坛首页并确认年龄
	if err = browser.NavigateTo(BaseURL); err != nil {
		log.Printf("导航到首页失败: %v", err)
		scheduleRetry("导航到首页失败: " + err.Error())
		return
	}
	if err = browser.ConfirmAge(); err != nil {
		log.Printf("确认年龄失败: %v", err)
		scheduleRetry("确认年龄失败: " + err.Error())
		return
	}

	// 步骤2: 检查登录状态
	if err = browser.CheckLoginStatus(); err != nil {
		log.Printf("登录状态检查失败: %v", err)
		scheduleRetry("登录状态检查失败: " + err.Error())
		return
	}

	// 步骤3：访问论坛回帖页面
	if err = browser.NavigateTo(BaseURL + ReplySection); err != nil {
		log.Printf("导航到回帖页面失败: %v", err)
		scheduleRetry("导航到回帖页面失败: " + err.Error())
		return
	}

	// 步骤4：随机选择一个帖子进行回帖
	replyInfo, err := browser.ReplyToPost()
	if err != nil {
		log.Printf("回帖失败: %v", err)
		scheduleRetry("回帖失败: " + err.Error())
		return
	}

	// 步骤5：执行签到
	checkInResult, err := browser.CheckIn()
	if err != nil {
		log.Printf("签到失败: %v", err)
		scheduleRetry("签到失败: " + err.Error())
		return
	}

	// 步骤6：获取金币信息
	coins, err := browser.GetMyCoins()
	if err != nil {
		log.Printf("获取金币信息失败: %v", err)
		scheduleRetry("获取金币信息失败: " + err.Error())
		return
	}
	// 构建金币信息
	coinsInfo := fmt.Sprintf("当前金币: %d", coins)

	// 步骤7：发送通知
	notificationMsg := fmt.Sprintf(
		"✅ 98tang ✅，时间: %s\n%s\n%s\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		replyInfo,
		checkInResult,
		coinsInfo,
	)
	if err := SendTelegramNotification(notificationMsg); err != nil {
		log.Printf("发送通知失败: %v", err)
		scheduleRetry("发送通知失败: " + err.Error())
		return
	}

	// 任务成功，更新上次成功时间
	taskMutex.Lock()
	lastSuccessTime = time.Now()
	taskMutex.Unlock()
}

// scheduleRetry 安排任务重试
func scheduleRetry(reason string) {
	log.Printf("任务失败，原因: %s，将在 %v 后重试", reason, RetryInterval)

	// 取消之前的重试计时器（如果存在）
	if retryTimer != nil {
		retryTimer.Stop()
	}

	// 设置新的重试计时器
	retryTimer = time.AfterFunc(RetryInterval, func() {
		log.Println("开始重试任务...")
		executeTask()
	})

	// 发送失败通知
	failureMsg := fmt.Sprintf(
		"❌ 任务失败 ❌\n时间: %s\n原因: %s\n将在 %d 分钟后重试",
		time.Now().Format("2006-01-02 15:04:05"),
		reason,
		int(RetryInterval.Minutes()),
	)

	if err := SendTelegramNotification(failureMsg); err != nil {
		log.Printf("发送失败通知失败: %v", err)
	}
}

// startScheduler 启动定时调度器
func startScheduler() {
	scheduler = cron.New(cron.WithSeconds())

	// 添加定时任务
	_, err := scheduler.AddFunc(CronSchedule, executeTask)
	if err != nil {
		log.Fatalf("添加定时任务失败: %v", err)
	}

	// 启动调度器
	scheduler.Start()
}

// NewBrowser 创建一个新的 Browser 实例
func NewBrowser() (*Browser, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		// 非无头模式便于调试, 本地测试改成false，启动图形界面
		chromedp.Flag("headless", EnableHeadless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.NoFirstRun,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// 创建分配器上下文
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	// 创建 Chrome 上下文
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// 启动浏览器（空任务），确保 ctx 正常启动
	if err := chromedp.Run(ctx); err != nil {
		cancelCtx()
		cancelAlloc()
		return nil, err
	}
	// 合并取消函数
	combinedCancel := func() {
		cancelCtx()
		cancelAlloc()
	}
	return &Browser{
		ctx:    ctx,
		cancel: combinedCancel,
	}, nil
}

// 确认满18岁
func (b *Browser) ConfirmAge() error {
	// 等待年龄确认按钮可见
	if err := b.WaitForElement(ConfirmAgeSelector); err != nil {
		return err
	}
	// 点击确认按钮
	if err := b.Click(EnterButtonSelector); err != nil {
		return err
	}
	return nil
}

// 检查登陆状态是否有效，若无效则执行登陆并加载cookie
func (b *Browser) CheckLoginStatus() error {
	// 等待 Header 可见
	if err := b.WaitForElement(HeaderSelector); err != nil {
		log.Println("Header 元素不可见")
		return err
	}

	// 获取 Header 的 HTML 内容
	headerHTML, err := b.GetHTML(HeaderSelector)
	if err != nil {
		log.Println("获取 Header HTML 失败")
		return err
	}

	// 首先检查是否已经处于登录状态
	if strings.Contains(headerHTML, "退出") {
		log.Println("已处于登录状态")
		return nil
	}

	// 检查 cookies 文件是否存在且未过期（不超过7天）
	cookiesExpired := false
	cookiesExist := false

	// 检查 cookies 文件是否存在
	fileInfo, err := os.Stat("./cookies")
	if err != nil {
		// cookies 文件不存在或无法访问
		log.Printf("cookies 文件不存在或无法访问: %v", err)
		cookiesExist = false
	} else {
		// cookies 文件存在
		if fileInfo.Size() == 0 {
			cookiesExist = false
		} else {
			cookiesExist = true

			// 检查 cookies 文件的修改时间，如果超过7天则视为过期
			if time.Since(fileInfo.ModTime()).Hours() > 24*7 {
				log.Printf("cookies 已过期（超过7天），标记为过期")
				cookiesExpired = true
			}
		}
	}

	// 如果有有效的 cookies 文件，尝试使用它登录
	if cookiesExist && !cookiesExpired {
		// 使用 cookies 登录
		if err := b.SetCookies(); err != nil {
			log.Printf("使用 cookies 登录失败: %v", err)
		} else {
			// 等待页面重载
			time.Sleep(3 * time.Second)

			// 检查是否登录成功
			headerHTML, err := b.GetHTML(HeaderSelector)
			if err == nil && strings.Contains(headerHTML, "退出") {
				log.Println("使用 cookies 登录成功")
				return nil
			} else {
				log.Println("使用 cookies 登录失败，将尝试用户名密码登录")
			}
		}
	} else if cookiesExpired {
		// 如果 cookies 过期，删除文件
		log.Println("cookies 已过期，将删除文件")
		if err := os.Remove("./cookies"); err != nil {
			log.Printf("删除过期 cookies 文件失败: %v", err)
		} else {
			log.Println("已删除过期 cookies 文件")
		}
	}

	// 如果不是已登录状态，且 cookies 登录失败或没有有效的 cookies，则使用用户名密码登录
	if err := b.Login(); err != nil {
		log.Printf("用户名密码登录失败: %v", err)
		return err
	}

	// 登录成功后，保存 cookies 到文件
	cookiesFile, err := b.SaveCookies()
	if err != nil {
		log.Printf("保存 cookies 失败: %v", err)
		return err
	}

	log.Printf("登录成功，cookies 已保存到 %s", cookiesFile)
	return nil
}

// 登陆函数
func (b *Browser) Login() error {
	// 等待登陆表单可见
	if err := b.WaitForElement(LoginFormSelector); err != nil {
		log.Println("登陆表单不可见")
		return err
	}

	// 输入用户名和密码
	if err := b.Input(UsernameSelector, FormUsername); err != nil {
		log.Println("输入用户名失败")
		return err
	}
	if err := b.Input(PasswordSelector, FormPassword); err != nil {
		log.Println("输入密码失败")
		return err
	}
	// 点击登陆按钮
	if err := b.Click(LoginButtonSelector); err != nil {
		log.Println("点击登陆按钮失败")
		return err
	}

	time.Sleep(5 * time.Second)

	// 等待二步认证框可见
	if err := b.WaitForElement(TwoFactorSelector); err != nil {
		log.Println("二步认证框不可见")
		return err
	}

	// 输入二步认证的安全问题和答案
	err := b.Execute(
		chromedp.Evaluate(`
            document.querySelector('[id^="loginquestionid_"]').value = '`+SecurityQuestion+`';
        `, nil),
	)
	if err != nil {
		log.Println("设置安全问题失败:", err)
		return err
	}

	// 输入安全问题答案
	if err := b.Input(TwoFactorAnswerSelector, SecurityAnswer); err != nil {
		log.Println("输入二步认证答案失败")
		return err
	}
	if err := b.Click(TwoFactorButtonSelector); err != nil {
		log.Println("点击二步认证按钮失败")
		return err
	}

	time.Sleep(5 * time.Second)

	return nil
}

// SaveCookies 登陆后保存cookies到文件
func (b *Browser) SaveCookies() (string, error) {
	// 使用写入模式打开，并清空原文件内容
	file, err := os.OpenFile("./cookies", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Printf("打开cookies文件失败: %v", err)
		return "", err
	}
	defer file.Close()

	// 登录后等待页面切换，等待 header 中出现“退出”
	if err := b.WaitForElement(HeaderSelector); err != nil {
		log.Println("登陆后 Header 元素不可见")
		return "", err
	}
	// 确保 Header 中包含“退出”字样
	headerHTML, err := b.GetHTML(HeaderSelector)
	if err != nil {
		log.Println("登陆后获取 Header HTML 失败")
		return "", err
	}
	if !strings.Contains(headerHTML, "退出") {
		log.Println("登陆后 Header 中不包含“退出”字样")
		return "", err
	}
	// 获取 cookies
	err = b.Execute(
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				log.Printf("获取 cookies 失败: %v", err)
				return err
			}

			j, err := json.Marshal(cookies)
			if err != nil {
				return err
			}

			// 写入 JSON 数据到文件
			_, err = file.Write(j)
			return err
		}),
	)

	if err != nil {
		log.Fatal("cookies保存失败: ", err)
	}

	return file.Name(), nil
}

// SetCookies 读取Cookies文件并自动登录
func (b *Browser) SetCookies() error {
	var text string
	return b.Execute(
		chromedp.ActionFunc(func(ctx context.Context) error {
			file, err := os.Open("./cookies")
			if err != nil {
				return err
			}
			defer file.Close()

			// 读取文件数据
			jsonBlob, err := io.ReadAll(file)
			if err != nil {
				return err
			}

			var cookies []*network.CookieParam
			// Json解码
			err = json.Unmarshal(jsonBlob, &cookies)
			if err != nil {
				return err
			}
			err = network.SetCookies(cookies).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
		chromedp.Reload(),
		chromedp.Title(&text),
	)
}

// ReplyToPost 回帖函数
func (b *Browser) ReplyToPost() (string, error) {
	// 等待回帖内容可见
	if err := b.WaitForElement(ReplyContentsSelector); err != nil {
		log.Println("回帖内容不可见")
		return "", nil
	}

	// 定义一个结构体来存储帖子信息
	type ThreadInfo struct {
		ID    string `json:"id"`    // 帖子ID
		Title string `json:"title"` // 帖子标题
	}

	// 使用JavaScript获取所有以normalthread_开头的元素ID和标题
	var threadInfos []ThreadInfo
	err := b.Execute(
		chromedp.Evaluate(`
            (function() {
                // 查找所有ID以normalthread_开头的元素
                const elements = document.querySelectorAll('[id^="normalthread_"]');
                const threads = [];
                
                elements.forEach(el => {
                    // 获取元素ID
                    const id = el.id;
                    
                    // 查找标题元素 (class为"s xst"的a标签)
                    const titleEl = el.querySelector('a.s.xst');
                    const title = titleEl ? titleEl.textContent : '无标题';
                    
                    // 将信息添加到结果数组
                    threads.push({
                        id: id,
                        title: title
                    });
                });
                
                return threads;
            })()
        `, &threadInfos),
	)
	if err != nil || len(threadInfos) == 0 {
		log.Printf("获取帖子列表失败或帖子列表为空: %v", err)
		return "", nil
	}

	// 排除前4个帖子
	if len(threadInfos) > 4 {
		threadInfos = threadInfos[4:]
	} else {
		log.Printf("帖子数量少于4个，无法排除，使用所有帖子")
	}

	// 确保还有可用的帖子
	if len(threadInfos) == 0 {
		return "", fmt.Errorf("排除前4个帖子后没有可回复的帖子")
	}

	// 随机选择一个帖子
	randomIndex := rand.IntN(len(threadInfos))
	selectedThread := threadInfos[randomIndex]
	selectedThreadID := strings.TrimPrefix(selectedThread.ID, "normalthread_")

	// 构建帖子URL并访问
	threadURL := fmt.Sprintf("%s/forum.php?mod=viewthread&tid=%s", BaseURL, selectedThreadID)
	if err := b.NavigateTo(threadURL); err != nil {
		log.Printf("访问帖子页面失败: %v", err)
		return "", err
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 等待回帖输入框可见
	if err := b.WaitForElement(ReplyInputSelector); err != nil {
		log.Println("回帖输入框不可见")
		return "", err
	}

	// 随机选择回帖内容
	replyContent := ReplyContents[rand.IntN(len(ReplyContents))]
	// 输入回帖内容
	if err := b.Input(ReplyInputSelector, replyContent); err != nil {
		log.Println("输入回帖内容失败")
		return "", err
	}

	// 点击提交按钮
	if err := b.Click(ReplyButtonSelector); err != nil {
		log.Println("点击回帖提交按钮失败")
		return "", err
	}

	// 等待提交完成
	time.Sleep(5 * time.Second)

	replyInfo := fmt.Sprintf("成功回复帖子: \nID：%s, \n标题：%s, \n回帖：%s", selectedThreadID, selectedThread.Title, replyContent)
	log.Println(replyInfo)

	return replyInfo, nil

}

// 签到函数
func (b *Browser) CheckIn() (string, error) {
	// 访问签到页面
	if err := b.NavigateTo(BaseURL + CheckInSection); err != nil {
		log.Println("导航签到页失败")
		return "", err
	}

	// 检查签到按钮是否存在
	exists, err := b.ElementExists(CheckInButtonSelector)
	if err != nil {
		log.Println("检查签到按钮失败")
		return "", err
	}
	if !exists {
		// 检查是否已经签到
		checkInDoneExists, err := b.ElementExists(CheckInDoneSelector)
		if err != nil {
			log.Println("检查签到结果失败")
			return "", err
		}
		if checkInDoneExists {
			// 获取签到结果
			if err := b.WaitForElement(CheckInDoneSelector); err != nil {
				log.Println("签到结果不可见")
				return "", err
			}
			// 获取签到结果的HTML
			checkInResultHTML, err := b.GetHTML(CheckInDoneSelector)
			if err != nil {
				log.Println("获取签到结果HTML失败")
				return "", err
			}

			// 签到结果
			var resultText string
			if strings.Contains(checkInResultHTML, "今日已签到") {
				resultText = "98堂每日签到成功，获得2金钱"
			} else {
				resultText = "98堂每日签到签到失败,进行重试"
			}

			return resultText, nil
		}
	}

	// 点击签到按钮
	if err := b.Click(CheckInButtonSelector); err != nil {
		log.Println("点击签到按钮失败")
		return "", err
	}

	// 等待签到验证框可见
	if err := b.WaitForElement(CheckInVerifyFormSelector); err != nil {
		log.Println("签到验证框不可见")
		return "", err
	}

	// 等待验证问题加载完成
	time.Sleep(5 * time.Second)

	// 获取 CheckInVerifyFormSelector 的 HTML 内容
	verifyFormHTML, err := b.GetHTML(CheckInVerifyFormSelector)
	if err != nil || verifyFormHTML == "" {
		log.Println("获取签到验证框 HTML 失败 或 HTML 为空")
		return "", err
	}

	// 获取输入框ID和问题文本直接使用JavaScript
	var result struct {
		InputID      string `json:"inputId"`
		QuestionText string `json:"questionText"`
	}

	err = b.Execute(
		chromedp.Evaluate(`
            (function() {
                // 查找验证问答表单
                const form = document.querySelector('.fwin form');
                if (!form) return {inputId: "", questionText: ""};
                
                // 查找输入框
                const input = form.querySelector('input[name="secanswer"]');
                const inputId = input ? input.id : "";
                
                // 查找问题文本 - 位于<br>标签之后的文本节点
                let questionText = "";
                const td = form.querySelector('td');
                if (td) {
                    // 获取TD中的HTML
                    const html = td.innerHTML;
                    // 使用正则表达式提取<br>后面的文本
                    const match = html.match(/<br[^>]*>(.*?)(<\/|$)/);
                    if (match && match[1]) {
                        questionText = match[1].trim();
                    }
                }
                
                return {
                    inputId: inputId,
                    questionText: questionText
                };
            })()
        `, &result),
	)

	if err != nil {
		log.Println("获取验证问题失败:", err)
		return "", err
	}

	// 如果没有获取到问题或输入框ID，尝试使用goquery
	if result.QuestionText == "" || result.InputID == "" {
		// 获取整个验证框的HTML
		verifyFormHTML, err := b.GetHTML("#fwin_pc_click_ddsign")
		if err != nil {
			log.Println("获取验证框HTML失败:", err)
			return "", err
		}

		log.Println("验证框HTML:", verifyFormHTML)

		// 使用goquery解析
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(verifyFormHTML))
		if err != nil {
			log.Println("解析HTML失败:", err)
			return "", err
		}

		// 查找输入框ID
		if result.InputID == "" {
			doc.Find("input[name='secanswer']").Each(func(i int, s *goquery.Selection) {
				if id, exists := s.Attr("id"); exists {
					result.InputID = id
				}
			})
		}

		// 查找问题文本
		if result.QuestionText == "" {
			doc.Find("td").Each(func(i int, s *goquery.Selection) {
				html, _ := s.Html()
				if strings.Contains(html, "<br>") {
					parts := strings.Split(html, "<br>")
					if len(parts) > 1 {
						cleanText := strings.TrimSpace(parts[1])
						// 移除HTML标签
						cleanText = strings.Split(cleanText, "<")[0]
						result.QuestionText = strings.TrimSpace(cleanText)
					}
				}
			})
		}
	}

	if result.QuestionText == "" {
		return "", fmt.Errorf("无法获取验证问题")
	}

	if result.InputID == "" {
		return "", fmt.Errorf("无法获取输入框ID")
	}

	// 解析问题并计算答案
	answer := ""

	// 处理减法问题
	if strings.Contains(result.QuestionText, "-") && strings.Contains(result.QuestionText, "=") {
		parts := strings.Split(result.QuestionText, "=")[0]
		numbers := strings.Split(parts, "-")
		if len(numbers) == 2 {
			a, errA := strconv.Atoi(strings.TrimSpace(numbers[0]))
			b, errB := strconv.Atoi(strings.TrimSpace(numbers[1]))

			if errA == nil && errB == nil {
				answer = strconv.Itoa(a - b)
				log.Printf("计算: %d - %d = %s", a, b, answer)
			} else {
				log.Printf("数字解析错误: %v, %v", errA, errB)
				return "", fmt.Errorf("无法解析计算表达式")
			}
		}
	} else if strings.Contains(result.QuestionText, "+") && strings.Contains(result.QuestionText, "=") {
		// 处理加法问题
		parts := strings.Split(result.QuestionText, "=")[0]
		numbers := strings.Split(parts, "+")
		if len(numbers) == 2 {
			a, errA := strconv.Atoi(strings.TrimSpace(numbers[0]))
			b, errB := strconv.Atoi(strings.TrimSpace(numbers[1]))

			if errA == nil && errB == nil {
				answer = strconv.Itoa(a + b)
				log.Printf("计算: %d + %d = %s", a, b, answer)
			}
		}
	}

	if answer == "" {
		return "", fmt.Errorf("无法计算答案")
	}

	// 输入验证答案
	if err := b.Input("#"+result.InputID, answer); err != nil {
		log.Println("输入验证答案失败:", err)
		return "", err
	}

	// 点击提交按钮
	if err := b.Click(CheckInVerifyButtonSelector); err != nil {
		log.Println("点击签到确认按钮失败")
		return "", err
	}

	// 导航到签到页面
	if err := b.NavigateTo(BaseURL + CheckInSection); err != nil {
		log.Println("导航签到页失败")
		return "", err
	}

	// 获取签到结果
	if err := b.WaitForElement(CheckInDoneSelector); err != nil {
		log.Println("签到结果不可见")
		return "", err
	}
	// 获取签到结果的HTML
	checkInResultHTML, err := b.GetHTML(CheckInDoneSelector)
	if err != nil {
		log.Println("获取签到结果HTML失败")
		return "", err
	}

	// 签到结果
	var resultText string
	if strings.Contains(checkInResultHTML, "今日已签到") {
		resultText = "98堂每日签到成功，获得2金钱"
	} else {
		resultText = "98堂每日签到签到失败,进行重试"
	}

	return resultText, nil
}

// GetMyCoins 查看我的金币
func (b *Browser) GetMyCoins() (int, error) {
	// 访问我的金币页面
	if err := b.NavigateTo(BaseURL + CoinsInfoSection); err != nil {
		log.Println("导航金币页失败")
		return 0, err
	}
	// 等待金币区域可见
	if err := b.WaitForElement(MyCoinsSelector); err != nil {
		log.Println("金币区域不可见")
		return 0, err
	}
	// 获取金币区域的HTML
	myCoinsHTML, err := b.GetHTML(MyCoinsSelector)
	if err != nil {
		log.Println("获取金币区域HTML失败")
		return 0, err
	}

	// 解析HTML获取金钱数量
	var coins int
	err = b.Execute(
		chromedp.Evaluate(`
            (function() {
                const coinElement = document.querySelector('.xi1.cl');
                if (!coinElement) return 0;
                
                // 获取innerHTML并提取金钱数值
                const html = coinElement.innerHTML;
                const match = html.match(/金钱: <\/em>(\d+)/);
                if (match && match[1]) {
                    return parseInt(match[1], 10);
                }
                return 0;
            })()
        `, &coins),
	)

	if err != nil {
		log.Printf("解析金钱数据失败: %v", err)
		return 0, err
	}

	// 如果JavaScript方法没有提取到，尝试使用goquery解析
	if coins == 0 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(myCoinsHTML))
		if err != nil {
			log.Println("使用goquery解析金币HTML失败:", err)
			return 0, err
		}

		// 查找金钱值
		doc.Find("em").Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), "金钱:") {
				// 获取元素的父节点(li)的文本
				parentText := s.Parent().Text()
				// 提取数字
				re := regexp.MustCompile(`金钱:\s*(\d+)`)
				match := re.FindStringSubmatch(parentText)
				if len(match) >= 2 {
					coins, _ = strconv.Atoi(match[1])
				}
			}
		})
	}

	if coins == 0 {
		log.Println("无法获取金币数量")
		return 0, fmt.Errorf("无法获取金币数量")
	}
	return coins, nil
}

// SendTelegramNotification 发送 Telegram 消息通知
func SendTelegramNotification(message string) error {
	bot, err := tgbotapi.NewBotAPI(MyBotToken)
	if err != nil {
		log.Printf("创建 Telegram Bot 实例失败: %v", err)
		return err
	}
	bot.Debug = false

	// 构建发送消息对象
	msg := tgbotapi.NewMessage(ChatID, message)
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("发送 Telegram 消息通知失败: %v", err)
		return err
	}
	return nil

}

// // main 函数
// func main() {
// 	// 随机睡眠
// 	delay := rand.IntN(WaitingTime)
// 	log.Printf("等待 %d 秒后开始执行", delay)
// 	time.Sleep(time.Duration(delay) * time.Second)

// 	// 创建浏览器实例
// 	browser, err := NewBrowser()
// 	if err != nil {
// 		log.Fatalf("无法创建浏览器实例: %v", err)
// 	}
// 	defer browser.Close()

// 	// 1. 访问论坛首页，并确认年龄
// 	if err = browser.NavigateTo(BaseURL); err != nil {
// 		log.Printf("导航首页失败: %v", err)
// 		return
// 	}
// 	if err = browser.ConfirmAge(); err != nil {
// 		log.Printf("确认年龄失败: %v", err)
// 		return
// 	}

// 	// 2. 检查登陆状态
// 	if err = browser.CheckLoginStatus(); err != nil {
// 		log.Printf("检查登录状态失败: %v", err)
// 		return
// 	}

// 	// 3. 访问论坛回帖页面
// 	replyURL := BaseURL + ReplySection
// 	if err = browser.NavigateTo(replyURL); err != nil {
// 		log.Printf("导航回帖页失败: %v", err)
// 		return
// 	}

// 	// 4. 进行回帖
// 	replyInfo, err := browser.ReplyToPost()
// 	if err != nil {
// 		log.Printf("回帖失败: %v", err)
// 		return
// 	}

// 	currentTime := time.Now().Format("2006年01月02日 15:04:05")

// 	// 5. 进行签到
// 	checkInResult, err := browser.CheckIn()
// 	if err != nil {
// 		log.Printf("签到失败: %v", err)
// 		return
// 	}

// 	// 6. 获取签到后的金钱数量
// 	coins, err := browser.GetMyCoins()
// 	if err != nil {
// 		log.Printf("获取金币数量失败: %v", err)
// 		return
// 	}
// 	// 构建金币信息
// 	coinsInfo := fmt.Sprintf("当前金币数量: %d", coins)

// 	// 7. 发送 Telegram 通知
// 	message := fmt.Sprintf("%s\n%s\n%s\n%s", currentTime, replyInfo, checkInResult, coinsInfo)
// 	if err = SendTelegramNotification(message); err != nil {
// 		log.Printf("发送 Telegram 通知失败: %v", err)
// 		return
// 	}
// 	log.Printf("Telegram 通知发送成功: %s", message)
// 	// 7. 关闭浏览器
// 	browser.Close()
// }

func main() {
	log.Println("程序启动...")

	// 启动调度器
	startScheduler()

	// 如果配置了立即执行任务，则立即执行一次
	if RunOnStart {
		go executeTask()
	}

	// 保持程序运行
	log.Println("程序已启动，按Ctrl+C停止")

	// 防止程序退出
	select {}
}
