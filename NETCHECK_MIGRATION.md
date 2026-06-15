# NetCheck.bat 迁移到 Go 客户端

## 概述

NetCheck.bat 的完整功能已集成到 Go 客户端中，通过 `guard` 命令启动守护模式。

## 功能对比

| 功能 | NetCheck.bat | Go 客户端 (guard 模式) |
|------|-------------|----------------------|
| 进程检测 | `tasklist /FI "IMAGENAME eq sdp.exe"` | `isSDPProcessRunning()` |
| 网络校验 | `curl -s -m 3 http://www.msftconnecttest.com/connecttest.txt` | `checkApplicationLayerNetwork()` |
| 修复操作 | `start "" /min /d "D:\SDP\ztgClient\AccInject" ztgLoader.exe -u` | `stopWFPService()` |
| 日志记录 | `echo [%date% %time%] >> run_log.txt` | `writeGuardLog()` |
| 循环间隔 | `timeout /t 5` | `time.Sleep(5 * time.Second)` |
| 进程等待 | `timeout /t 10` | `time.Sleep(10 * time.Second)` |
| 修复冷却 | `timeout /t 3` | `time.Sleep(3 * time.Second)` |

## 关键改进

### 1. 自动路径定位
- NetCheck.bat 硬编码 `D:\SDP\ztgClient\AccInject`
- Go 客户端自动扫描所有盘符 (C-Z)，智能定位 SDP 安装目录

### 2. 进程检测更精准
- 使用 Go 原生 `exec.Command` 执行 tasklist
- 解析输出判断进程是否存在
- 窗口隐藏，无闪烁

### 3. HTTP 网络校验
- 使用标准 HTTP 客户端，3秒超时
- 禁用 keep-alive 避免连接池问题
- 检查响应内容是否包含 "Microsoft Connect Test"

### 4. 详细日志系统
- 时间戳格式：`2006-01-02 15:04:05`
- 保存位置：`%PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt`
- 记录启动、断网检测、修复执行、累计次数

### 5. 集成审批系统
- 守护进程启动前检查本地审批状态
- 未审批设备无法启动守护模式
- 保留原有的设备管理和黑名单功能

## 使用方式

### 手动启动守护进程

```powershell
Fuck0TrustClient.exe guard
```

### 安装计划任务（开机自启）

```powershell
Fuck0TrustClient.exe install-task
```

计划任务会在用户登录时自动启动守护进程，无需手动干预。

## 配置参数

所有参数在 `main.go` 的常量中定义，可根据需要调整：

```go
const (
    SDPProcessName          = "sdp.exe"
    MSFTConnectTestURL      = "http://www.msftconnecttest.com/connecttest.txt"
    MSFTConnectTestKeyword  = "Microsoft Connect Test"
    GuardLoopInterval       = 5 * time.Second
    ProcessCheckWaitTime    = 10 * time.Second
    FixCooldownTime         = 3 * time.Second
)
```

## 迁移步骤

1. **停止旧的 NetCheck.bat**
   - 手动关闭正在运行的 NetCheck.bat 窗口
   - 删除相关的计划任务（如果有）

2. **部署 Go 客户端**
   - 下载或编译 `Fuck0TrustClient.exe`
   - 放置到合适的目录

3. **提交审批**
   - 运行客户端，填写联系方式
   - 点击"提交审批"，等待管理员通过

4. **同步审批状态**
   - 审批通过后，点击"同步审批状态"
   - 确认显示"已通过"

5. **安装计划任务**
   - 右键以管理员身份运行客户端
   - 点击"安装计划任务"
   - 或命令行：`Fuck0TrustClient.exe install-task`

6. **验证运行**
   - 重启电脑，检查守护进程是否自动启动
   - 查看日志文件：`%PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt`

## 日志示例

```
[2024-01-15 10:30:00] 守护进程启动
[2024-01-15 10:35:12] 真实断网，执行修复程序，累计次数：1
[2024-01-15 10:35:15] 修复执行成功
[2024-01-15 11:20:45] 真实断网，执行修复程序，累计次数：2
[2024-01-15 11:20:48] 修复执行成功
```

## 故障排查

### 守护进程无法启动
- 检查是否已通过审批：`Fuck0TrustClient.exe status`
- 查看错误提示信息

### 日志文件不存在
- 确认 `%PROGRAMDATA%` 目录权限
- 手动创建目录：`%PROGRAMDATA%\Fuck0TrustApprovalClient`

### 修复操作执行失败
- 检查 SDP 是否正确安装
- 确认 `ztgLoader.exe` 和 `AccInject10_x64.sys` 存在
- 以管理员权限运行

### 计划任务未自动运行
- 检查计划任务是否创建：运行 `taskschd.msc`
- 查找任务名称：`Fuck0Trust_Status_Check`
- 确认触发器设置为"登录时"
- 确认操作为：`"路径\Fuck0TrustClient.exe" guard`

## 优势总结

1. **单一可执行文件**：无需依赖 curl，无需外部脚本
2. **体积小**：3-6 MB，集成所有功能
3. **稳定性高**：编译型语言，类型安全
4. **易于管理**：图形界面 + 命令行，双重操作方式
5. **权限控制**：集成审批系统，设备黑名单管理
6. **跨盘符支持**：自动扫描定位 SDP 安装目录
7. **详细日志**：时间戳、累计次数、操作结果
8. **无窗口闪烁**：后台静默运行，不干扰用户
