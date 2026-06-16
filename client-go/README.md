# Windows 客户端 (Go 重构版)

这是使用 Go 语言重构的安全合规版客户端，功能与 Python 版本完全一致，但打包体积减少 70-80%。

## 主要改进

- ✅ **体积大幅减少**: 从 15-25 MB 减少到 3-6 MB
- ✅ **原生二进制**: 无需 Python 运行时，启动更快
- ✅ **UI 优化**: 改进布局和交互体验，窗口更紧凑（280×300）
- ✅ **更好的网络处理**: 连接复用和自动重试
- ✅ **类型安全**: Go 编译时类型检查，更稳定
- ✅ **嵌入图标**: app.ico 自动嵌入到可执行文件中

## 功能特性

- 生成当前设备唯一 ID
- 向 Cloudflare Worker 提交审批申请
- 查询审批状态
- 审批通过后，仅查询 `WFPRedirect` 服务状态
- 审批通过后，可写入一个每 4 分钟执行一次的状态检查计划任务

> 注意：本客户端不会停止、禁用或绕过安全/零信任驱动。

## 构建

### 方式 1: GitHub Actions（推荐）

代码已包含自动构建工作流 `.github/workflows/build-go-client.yml`。

**使用方法：**
1. 推送代码到 GitHub
2. 自动触发构建（或在 Actions 页面手动触发）
3. 在 Actions 运行页面下载 `Fuck0TrustClient-windows` 构建产物
4. 构建产物保留 90 天

**触发条件：**
- 推送到 `main`/`master` 分支且修改了 `client-go/**` 文件
- 手动触发（Actions → Build Go Client → Run workflow）
- 创建 tag 时会自动创建 Release

### 方式 2: 本地构建

**前置要求：**
1. 安装 Go 1.21+: https://go.dev/dl/
2. (可选) 安装 rsrc: `go install github.com/akavel/rsrc@latest`
3. (可选) 安装 UPX: https://github.com/upx/upx/releases

**构建步骤：**
```bash
# Windows
.\build.bat

# 或手动构建
go mod download
go build -ldflags="-s -w -H=windowsgui" -o Fuck0TrustClient.exe .
```

## 使用方法

### GUI 模式（推荐）

直接双击 `Fuck0TrustClient.exe` 会打开图形界面：

1. 查看设备 ID 和当前审批状态
2. 填写申请备注（可选）
3. 点击"提交审批"
4. 管理员审批后点击"同步审批状态"
5. 审批通过后可点击"执行一次"或"安装计划任务"

### CLI 模式

```bash
# 提交审批申请
Fuck0TrustClient.exe request --note "申请说明"

# 查询审批状态
Fuck0TrustClient.exe status

# 执行一次受控功能（需先审批通过）
Fuck0TrustClient.exe run

# 安装计划任务（需管理员权限和审批通过）
Fuck0TrustClient.exe install-task

# 删除计划任务（需管理员权限）
Fuck0TrustClient.exe remove-task
```

## 配置文件

配置文件自动保存在：`%ProgramData%\Fuck0TrustApprovalClient\config.json`

包含：
- 设备审批状态缓存
- 请求提交时间戳
- API 地址配置

## 技术栈

- **语言**: Go 1.21+
- **GUI 框架**: lxn/walk (Windows 原生 GUI)
- **HTTP 客户端**: 标准库 net/http
- **配置存储**: JSON 文件
- **系统交互**: Windows Registry, Task Scheduler

## 与 Python 版本对比

| 特性 | Python 版 | Go 版 |
|------|-----------|-------|
| 打包体积 | ~20 MB | ~4 MB |
| 启动速度 | 较慢 | 快速 |
| 内存占用 | ~50 MB | ~15 MB |
| 依赖 | Python + requests + tkinter | 无外部依赖 |
| 交叉编译 | 困难 | 简单 |

## 开发

```bash
# 安装依赖
go mod download

# 运行（开发模式）
go run .

# 格式化代码
go fmt ./...

# 静态检查
go vet ./...
```

## 许可证

与项目主仓库保持一致
