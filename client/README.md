# Windows 客户端

这是安全合规版客户端：

- 生成当前设备唯一 ID；
- 向 Cloudflare Worker 提交审批申请；
- 查询审批状态；
- 审批通过后，仅查询 `WFPRedirect` 服务状态；
- 审批通过后，可写入一个每 4 分钟执行一次的状态检查计划任务。

> 注意：本客户端不会停止、禁用或绕过安全/零信任驱动。

## 使用

```powershell
Fuck0TrustClient.exe --api https://你的-worker.workers.dev request --note "申请说明"
Fuck0TrustClient.exe --api https://你的-worker.workers.dev status
Fuck0TrustClient.exe --api https://你的-worker.workers.dev run
Fuck0TrustClient.exe --api https://你的-worker.workers.dev install-task
Fuck0TrustClient.exe remove-task
```

第一次传入 `--api` 后会写入 `%ProgramData%\Fuck0TrustApprovalClient\config.json`，后续可以省略。
