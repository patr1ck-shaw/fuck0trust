# 故障排查指南

## 1. 程序闪退问题

### 症状
- 双击程序后立即闪退
- 没有任何窗口显示
- 没有错误提示

### 常见原因及解决方案

#### 原因 1：编译参数错误
**问题**：使用了 `-s -w` 参数去除符号表，导致程序无法正常运行

**解决方案**：
```bash
# 正确的编译命令
go build -tags walkgui -ldflags="-H=windowsgui" -o Fuck0TrustClient.exe .
```

**不要使用**：
```bash
# 错误 - 会导致闪退
go build -ldflags="-s -w -H=windowsgui" -o Fuck0TrustClient.exe .
```

#### 原因 2：CGO 未启用
**问题**：walk GUI 库依赖 CGO，未启用会导致闪退

**解决方案**：
```bash
set CGO_ENABLED=1
go build -tags walkgui -ldflags="-H=windowsgui" -o Fuck0TrustClient.exe .
```

#### 原因 3：缺少构建标签
**问题**：没有使用 `-tags walkgui` 标签，导致 GUI 代码未被编译

**解决方案**：
```bash
go build -tags walkgui -ldflags="-H=windowsgui" -o Fuck0TrustClient.exe .
```

#### 原因 4：walk 库空指针问题
**问题**：官方 lxn/walk 库存在空指针 bug

**解决方案**：使用社区修复版（已在 go.mod 中配置）
```go
replace github.com/lxn/walk => github.com/tailscale/walk v0.0.0-20240403170109-4e66d5f4cdc9
```

### 诊断步骤

1. **查看崩溃日志**
```powershell
type %TEMP%\fuck0trust_crash.log
```

2. **命令行测试**
```powershell
# 测试基本功能
Fuck0TrustClient.exe status

# 如果命令行正常但 GUI 闪退，说明是 GUI 相关问题
```

3. **检查依赖**
```powershell
cd client-go
go mod tidy
go mod download
```

## 2. GitHub Actions 构建问题

### 症状
- Actions 构建的程序打不开
- 本地编译正常，Actions 编译的程序闪退

### 解决方案

#### 确保使用正确的 Windows 版本
```yaml
runs-on: windows-2019  # 不要用 windows-latest
```

#### 确保正确设置环境变量
```yaml
- name: Build Windows Executable
  working-directory: client-go
  env:
    CGO_ENABLED: "1"
    GOOS: windows
    GOARCH: amd64
  run: go build -tags walkgui -ldflags="-H=windowsgui" -o Fuck0TrustClient.exe .
  shell: cmd
```

#### 不要使用 `-s -w` 参数
```yaml
# 错误 ❌
run: go build -ldflags="-s -w -H=windowsgui" ...

# 正确 ✅
run: go build -tags walkgui -ldflags="-H=windowsgui" ...
```

## 3. 守护进程问题

### 症状
- `guard` 命令启动后立即退出
- 日志文件没有生成

### 解决方案

#### 检查审批状态
```powershell
Fuck0TrustClient.exe status
```

守护进程需要审批通过才能启动。

#### 检查日志位置
```powershell
type %PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt
```

#### 手动测试守护功能
```powershell
# 先测试单次执行
Fuck0TrustClient.exe run

# 再测试守护模式
Fuck0TrustClient.exe guard
```

## 4. 计划任务问题

### 症状
- 计划任务创建失败
- 计划任务不自动运行

### 解决方案

#### 确保管理员权限
```powershell
# 右键选择"以管理员身份运行"
Fuck0TrustClient.exe install-task
```

#### 检查计划任务配置
```powershell
# 打开任务计划程序
taskschd.msc

# 查找任务名称
Fuck0Trust_Status_Check
```

#### 确认任务操作路径
任务应该调用：
```
"完整路径\Fuck0TrustClient.exe" guard
```

不是：
```
"完整路径\Fuck0TrustClient.exe" run
```

## 5. 网络检测问题

### 症状
- 守护进程运行但不执行修复
- 明明断网了但没有触发修复

### 解决方案

#### 检查 sdp.exe 进程
```powershell
tasklist | findstr sdp.exe
```

守护进程只在 sdp.exe 运行时才工作。

#### 测试网络检测
```powershell
# 测试微软连接测试 URL
curl http://www.msftconnecttest.com/connecttest.txt
```

应该返回：`Microsoft Connect Test`

#### 查看详细日志
```powershell
type %PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt
```

日志会显示：
- 守护进程启动时间
- 每次断网检测
- 修复执行结果
- 累计修复次数

## 6. SDP 路径问题

### 症状
- 修复功能执行失败
- 提示找不到 ztgLoader.exe

### 解决方案

#### 确认 SDP 安装位置
程序会自动扫描所有盘符 (C-Z) 查找：
```
X:\SDP\ztgClient\AccInject\
```

#### 手动指定路径（如果自动定位失败）
在代码中修改 `findSDPPath()` 函数的默认路径。

## 7. 编译环境问题

### Go 版本要求
```bash
go version
# 需要 Go 1.21 或更高版本
```

### 必需的编译工具

#### rsrc（可选，用于嵌入 manifest）
```bash
go install github.com/akavel/rsrc@latest
```

#### UPX（可选，用于压缩）
```bash
choco install upx
```

### 编译器要求
Windows 需要 MinGW-w64 或 TDM-GCC 来支持 CGO。

## 8. 快速验证清单

在报告问题前，请先完成以下检查：

- [ ] 使用正确的编译命令（包含 `-tags walkgui`）
- [ ] CGO 已启用 (`CGO_ENABLED=1`)
- [ ] 没有使用 `-s -w` 参数
- [ ] go.mod 包含 walk 库的 replace 指令
- [ ] 程序具有管理员权限（安装任务时）
- [ ] 设备已通过审批（守护模式时）
- [ ] SDP 已安装且路径正确
- [ ] 查看了崩溃日志和守护日志

## 9. 获取详细诊断信息

如果问题仍未解决，请提供以下信息：

1. **系统信息**
```powershell
systeminfo | findstr /C:"OS"
```

2. **Go 版本**
```bash
go version
```

3. **编译命令**
```bash
# 你使用的完整编译命令
```

4. **崩溃日志**
```powershell
type %TEMP%\fuck0trust_crash.log
```

5. **守护日志**
```powershell
type %PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt
```

6. **构建标签验证**
```bash
go list -tags walkgui -f '{{.GoFiles}}'
```

## 10. 已知问题及解决方案

### 问题：双击没反应
- **原因**：窗口被隐藏或最小化
- **解决**：代码中已添加强制显示窗口逻辑

### 问题：托盘图标不显示
- **原因**：app.ico 文件缺失
- **解决**：代码会自动使用系统默认图标

### 问题：Windows Defender 报毒
- **原因**：未签名的可执行文件
- **解决**：添加到白名单，或使用代码签名证书

### 问题：网络检测不准确
- **原因**：DNS 或代理干扰
- **解决**：程序使用 Microsoft Connect Test URL，绕过大部分代理
