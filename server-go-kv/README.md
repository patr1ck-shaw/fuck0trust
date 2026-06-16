# Fuck0Trust Go KV 服务器

基于 Go + BadgerDB 的轻量级设备审批服务器。

## 特性

- ✅ **极小镜像**：10-20MB（静态编译 + Alpine）
- ✅ **高性能**：BadgerDB 嵌入式 KV 存储，实时读写
- ✅ **无外部依赖**：数据库嵌入在应用内
- ✅ **完整 Web UI**：与 Cloudflare Worker 版本界面一致
- ✅ **零配置**：开箱即用

## 快速部署

### Docker 部署（推荐）

```bash
docker pull ghcr.io/patr1ck-shaw/fuck0trust-server:latest

docker run -d \
  --name fuck0trust-server \
  -p 3000:3000 \
  -e ADMIN_TOKEN="your-strong-token" \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  ghcr.io/patr1ck-shaw/fuck0trust-server:latest
```

### Docker Compose

```yaml
services:
  fuck0trust-server:
    image: ghcr.io/patr1ck-shaw/fuck0trust-server:latest
    container_name: fuck0trust-server
    ports:
      - "3000:3000"
    environment:
      - ADMIN_TOKEN=your-strong-token
      - PORT=3000
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

启动：
```bash
docker-compose up -d
```

## 配置

### 环境变量

- `ADMIN_TOKEN`：管理员密码（必须设置）
- `PORT`：服务端口（默认 3000）

### 数据目录

BadgerDB 数据存储在 `./data/badger/` 目录，挂载到容器 `/app/data`。

## 访问管理后台

```
http://your-server:3000/admin
```

输入 `ADMIN_TOKEN` 登录后即可审批设备。

## API 接口

与 Cloudflare Worker 版本完全兼容：

**客户端接口：**
- `GET /health` - 健康检查
- `POST /api/request` - 提交审批申请
- `GET /api/status?deviceId=...` - 查询审批状态

**管理接口：**
- `GET /admin` - Web 管理面板
- `GET /api/admin/devices` - 列出设备（需 Bearer Token）
- `POST /api/admin/approve` - 批准设备（需 Bearer Token）
- `POST /api/admin/deny` - 拒绝设备（需 Bearer Token）

管理 API 需要 Header：
```http
Authorization: Bearer <ADMIN_TOKEN>
```

## 本地开发

```bash
# 安装依赖
go mod download

# 运行
export ADMIN_TOKEN="test-token"
go run main.go
```

## 构建

```bash
# 本地构建
CGO_ENABLED=0 go build -ldflags="-w -s" -o fuck0trust-server .

# Docker 构建
docker build -t fuck0trust-server .
```

## 技术栈

- **语言**：Go 1.22
- **存储**：BadgerDB v4（嵌入式 KV 数据库）
- **镜像**：Alpine Linux（最小化）
- **架构**：单一二进制，静态编译

## 与 Worker 版本对比

| 特性 | Go + BadgerDB | Cloudflare Worker |
|------|---------------|-------------------|
| 镜像大小 | 10-20MB | N/A（无服务器）|
| 性能 | 极高 | 高 |
| 部署 | 自托管 | 全球分布式 |
| 成本 | 服务器成本 | 免费额度 |
| 适用场景 | 国内服务器 | 海外用户 |
