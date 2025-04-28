APP_NAME=daysign98tang
BUILD_DIR=build

.PHONY: all clean build build-all

# 默认目标
all: clean build-all

# 清理构建目录
clean:
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

# 构建当前平台版本
build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) main.go

# 构建所有平台版本
build-all: clean
	# Linux x86_64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME)_x86 main.go
	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME)_arm64 main.go
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME)_mac main.go
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME)_windows.exe main.go

# 打包目标
package:
	cd $(BUILD_DIR) && \
	tar -czf $(APP_NAME)_x86.tar.gz $(APP_NAME)_x86 && \
	tar -czf $(APP_NAME)_arm64.tar.gz $(APP_NAME)_arm64 && \
	tar -czf $(APP_NAME)_mac.tar.gz $(APP_NAME)_mac && \
	zip $(APP_NAME)_windows.zip $(APP_NAME)_windows.exe