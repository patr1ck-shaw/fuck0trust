# Fuck0Trust 设备审批项目（安全合规版）

本项目包含两部分：

- `worker/`：Cloudflare Workers + KV 后端，用于保存设备审批状态；
- `client/`：Windows 客户端，可由 GitHub Actions 打包成单文件 EXE。

## 重要说明

你提供的 BAT 会执行 `sc stop WFPRedirect`。该行为会停止疑似零信任/网络安全相关驱动，并且你希望将其打包后分发给所有人、定时每 4 分钟执行。出于安全原因，本项目不会实现自动停止、禁用或绕过安全/零信任组件的功能。

当前实现保留了你需要的“设备 ID + 审批 + KV 持久化 + GitHub Actions 打包 EXE + 计划任务”架构，但客户端的受控功能仅为查询 `WFPRedirect` 状态，便于审计和排障。

## 后端接口

Worker 提供以下接口：

- `POST /api/request`：客户端提交设备申请；
- `GET /api/status?deviceId=...`：客户端查询审批状态；
- `GET /api/admin/devices`：管理员列出设备；
- `POST /api/admin/approve`：管理员批准设备；
- `POST /api/admin/deny`：管理员拒绝设备。

管理员接口需要 Header：

```http
Authorization: Bearer <ADMIN_TOKEN>
```

## Cloudflare Workers 部署

进入 `worker/`：

```bash
npm install
npx wrangler login
npx wrangler kv namespace create DEVICE_APPROVAL_KV
```

将输出的 KV namespace id 填入 `worker/wrangler.toml`：

```toml
[[kv_namespaces]]
binding = "DEVICE_APPROVAL_KV"
id = "你的 KV namespace id"
```

建议使用 secret 设置管理员 token：

```bash
npx wrangler secret put ADMIN_TOKEN
npx wrangler deploy
```

## 管理员审批示例

列出设备：

```bash
curl -H "Authorization: Bearer <ADMIN_TOKEN>" \
  https://你的-worker.workers.dev/api/admin/devices
```

批准设备：

```bash
curl -X POST \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"deviceId":"设备ID"}' \
  https://你的-worker.workers.dev/api/admin/approve
```

拒绝设备：

```bash
curl -X POST \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"deviceId":"设备ID"}' \
  https://你的-worker.workers.dev/api/admin/deny
```

## 客户端使用

从 GitHub Actions 的 `Build Windows Client` 工作流下载 `Fuck0TrustClient.exe`。

提交审批：

```powershell
Fuck0TrustClient.exe --api https://你的-worker.workers.dev request --note "申请说明"
```

查询审批状态：

```powershell
Fuck0TrustClient.exe --api https://你的-worker.workers.dev status
```

审批通过后执行一次受控功能：

```powershell
Fuck0TrustClient.exe --api https://你的-worker.workers.dev run
```

审批通过后写入计划任务（需要管理员权限）：

```powershell
Fuck0TrustClient.exe --api https://你的-worker.workers.dev install-task
```

删除计划任务（需要管理员权限）：

```powershell
Fuck0TrustClient.exe remove-task
```

## 本地构建 EXE

```powershell
cd client
.\build.ps1
```

产物位于：

```text
client\dist\Fuck0TrustClient.exe
```
