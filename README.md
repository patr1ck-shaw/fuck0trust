# Fuck0Trust

Fuck0Trust 是一个基于 **Cloudflare Workers + KV** 的设备审批项目，包含：

- `worker/dashboard-worker.js`：可直接粘贴到 Cloudflare Dashboard 的 Worker 单文件代码；
- `client-go/`：Go 语言 Windows 客户端，支持提交审批、同步审批状态、本地保存授权、安装/删除计划任务。

## 重要说明

本项目保留“设备 ID + 审批 + KV 持久化 + Windows 客户端 + 计划任务”架构。

客户端当前的受控功能仅用于查询/演示受控流程，不实现自动停止、禁用或绕过安全/零信任组件的能力。

## Worker 部署方式：Cloudflare Dashboard 直接粘贴

当前 Worker 侧只需要一个文件：

```text
worker/dashboard-worker.js
```

不再需要 Wrangler/TypeScript 项目文件。

### 1. 创建 KV

在 Cloudflare Dashboard 中创建一个 KV namespace，名称可自定义。

### 2. 创建 Worker

进入 Cloudflare Dashboard：

```text
Workers & Pages -> Create Worker
```

然后把 `worker/dashboard-worker.js` 的全部内容粘贴进去。

### 3. 绑定 KV

在 Worker 的 Settings/Bindings 中添加 KV Namespace binding：

```text
Variable name: DEVICE_APPROVAL_KV
KV namespace: 你创建的 KV
```

### 4. 设置管理员 Token

在 Worker 的 Variables/Secrets 中添加：

```text
ADMIN_TOKEN=你的强随机管理员密码
```

建议使用 Secret，不要使用默认值。

### 5. 访问管理后台

部署完成后访问：

```text
https://你的域名/admin
```

输入 `ADMIN_TOKEN` 登录后即可审批设备。

## Worker 接口

客户端接口：

- `GET /health`：健康检查；
- `POST /api/request`：客户端提交设备审批申请；
- `GET /api/status?deviceId=...`：客户端查询审批状态。

管理后台页面：

- `GET /admin`：设备审批管理面板；
- `POST /admin/login`：管理员登录；
- `POST /admin/logout`：管理员退出；
- `POST /admin/decision`：批准/拒绝设备。

管理员 API：

- `GET /api/admin/devices`：列出设备；
- `POST /api/admin/approve`：批准设备；
- `POST /api/admin/deny`：拒绝设备。

管理员 API 需要 Header：

```http
Authorization: Bearer <ADMIN_TOKEN>
```

### 客户端行为

- 打开客户端后自动检测 `/health`；
- 网络不可达时会弹窗提示网络环境存在问题；
- 点击”提交审批”后会提交当前设备信息；
- 同一设备本地限制 24 小时内只能提交一次审批；
- 点击”同步审批状态”会从 Worker 获取审批状态；
- 审批通过后会在本地永久保存授权；
- 后续执行受控功能或计划任务时只检查本地授权，不再联网校验。

### 守护模式（NetCheck）

客户端集成了 NetCheck.bat 的完整功能：

1. **进程监测**：持续检查 `sdp.exe` 进程是否运行
2. **应用层网络校验**：使用 Microsoft Connect Test URL 进行真实网络连通性检测
3. **自动修复**：检测到断网时自动执行驱动卸载修复
4. **详细日志**：记录每次检测和修复操作，包含时间戳和累计次数
5. **5秒循环**：与原 NetCheck.bat 保持一致的检测频率

守护进程日志保存位置：

```text
%PROGRAMDATA%\Fuck0TrustApprovalClient\guard_log.txt
```

本地配置保存位置：

```text
%PROGRAMDATA%\Fuck0TrustApprovalClient\config.json
```

## 客户端命令行

图形界面：

```powershell
Fuck0TrustClient.exe
```

提交审批：

```powershell
Fuck0TrustClient.exe request --note "申请说明"
```

同步/查询审批状态：

```powershell
Fuck0TrustClient.exe status
```

审批通过后执行一次受控功能：

```powershell
Fuck0TrustClient.exe run
```

审批通过后启动守护进程（NetCheck 模式）：

```powershell
Fuck0TrustClient.exe guard
```

审批通过后安装计划任务（需要管理员权限）：

```powershell
Fuck0TrustClient.exe install-task
```

删除计划任务（需要管理员权限）：

```powershell
Fuck0TrustClient.exe remove-task
```

## 客户端构建

### GitHub Actions 自动构建（推荐）

1. 推送代码到 GitHub
2. 在 Actions 页面查看构建结果
3. 下载构建产物 `Fuck0TrustClient-windows`（约 3-6 MB）

工作流会自动：
- 编译 Go 代码
- 嵌入 Windows manifest
- UPX 压缩优化
- 上传构建产物

### 本地构建

```powershell
cd client-go
.\build.bat
```

详见：[client-go/README.md](client-go/README.md)

## 客户端特性

- ✅ **体积小**: 3-6 MB（使用 Go 原生编译 + UPX 压缩）
- ✅ **启动快**: 毫秒级启动，无需运行时
- ✅ **无依赖**: 单一可执行文件
- ✅ **类型安全**: 编译时检查，运行稳定
- ✅ **原生 GUI**: Windows 原生控件，体验流畅
