# Fuck0Trust 服务器 - Node.js 版本

基于 Node.js + Express + SQLite 的设备审批服务器，可部署在任何 Linux/Debian 机器上。

## 特性

- ✅ **轻量级**：Node.js + SQLite，无需复杂数据库
- ✅ **高性能**：SQLite WAL 模式 + Express
- ✅ **持久化**：数据存储在本地 SQLite 数据库
- ✅ **完整 API**：与 Cloudflare Worker 版本兼容
- ✅ **易部署**：支持 systemd、PM2、Docker

## 快速开始

### 1. 安装依赖

```bash
cd server-node
npm install
```

### 2. 配置环境变量

```bash
# 创建 .env 文件（可选）
export PORT=3000
export ADMIN_TOKEN="your-strong-random-token"
```

### 3. 启动服务

```bash
# 开发模式（自动重启）
npm run dev

# 生产模式
npm start
```

服务器将在 `http://0.0.0.0:3000` 启动。

## 生产部署

### 方式 1: systemd（推荐）

创建 `/etc/systemd/system/fuck0trust.service`：

```ini
[Unit]
Description=Fuck0Trust Approval Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/fuck0trust/server-node
Environment="NODE_ENV=production"
Environment="PORT=3000"
Environment="ADMIN_TOKEN=your-strong-token-here"
ExecStart=/usr/bin/node server.js
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable fuck0trust
sudo systemctl start fuck0trust
sudo systemctl status fuck0trust
```

查看日志：

```bash
sudo journalctl -u fuck0trust -f
```

### 方式 2: PM2

```bash
# 安装 PM2
npm install -g pm2

# 启动
pm2 start server.js --name fuck0trust

# 开机自启
pm2 startup
pm2 save

# 查看状态
pm2 status
pm2 logs fuck0trust
```

### 方式 3: Docker

创建 `Dockerfile`：

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY server.js ./
RUN mkdir -p data
EXPOSE 3000
CMD ["node", "server.js"]
```

构建并运行：

```bash
docker build -t fuck0trust-server .
docker run -d \
  --name fuck0trust \
  -p 3000:3000 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_TOKEN=your-token \
  --restart unless-stopped \
  fuck0trust-server
```

## Nginx 反向代理

```nginx
server {
    listen 80;
    server_name approval.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

配置 HTTPS（使用 Let's Encrypt）：

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d approval.yourdomain.com
```

## API 接口

### 客户端接口

#### 健康检查
```
GET /health
```

#### 提交审批申请
```
POST /api/request
Content-Type: application/json

{
  "deviceId": "abc123...",
  "hostname": "DESKTOP-XXX",
  "username": "user",
  "note": "联系方式或申请理由"
}
```

#### 查询审批状态
```
GET /api/status?deviceId=abc123...
```

### 管理员接口

所有管理员接口需要携带 `Authorization: Bearer <ADMIN_TOKEN>` 头。

#### 获取设备列表
```
GET /api/admin/devices
Authorization: Bearer <ADMIN_TOKEN>
```

#### 批准设备
```
POST /api/admin/approve
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json

{
  "deviceId": "abc123..."
}
```

#### 拒绝设备
```
POST /api/admin/deny
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json

{
  "deviceId": "abc123..."
}
```

#### 拉黑设备
```
POST /api/admin/blacklist
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json

{
  "deviceId": "abc123..."
}
```

#### 删除设备
```
DELETE /api/admin/device/:deviceId
Authorization: Bearer <ADMIN_TOKEN>
```

## 数据库

数据存储在 `data/approval.db`（SQLite）。

### 备份

```bash
# 备份
sqlite3 data/approval.db ".backup backup.db"

# 或使用 cp
cp data/approval.db data/approval.db.backup
```

### 查询数据

```bash
sqlite3 data/approval.db
sqlite> SELECT * FROM devices;
sqlite> .exit
```

## 客户端配置

修改客户端的 `APIBase` 为你的服务器地址：

```go
const APIBase = "https://approval.yourdomain.com"
```

重新编译客户端即可。

## 性能优化

1. **启用 SQLite WAL 模式**（默认已启用）
2. **使用 Nginx 反向代理** + gzip 压缩
3. **配置防火墙**：只开放必要端口
4. **监控**：使用 PM2 或 systemd 自动重启

## 安全建议

- ✅ 使用强随机 `ADMIN_TOKEN`
- ✅ 配置 HTTPS（Let's Encrypt）
- ✅ 定期备份数据库
- ✅ 配置防火墙规则
- ✅ 限制管理员接口访问（IP 白名单）

## 故障排查

### 查看日志

```bash
# systemd
sudo journalctl -u fuck0trust -f

# PM2
pm2 logs fuck0trust

# Docker
docker logs -f fuck0trust
```

### 端口占用

```bash
sudo lsof -i :3000
sudo netstat -tulpn | grep 3000
```

### 权限问题

确保数据库目录可写：

```bash
sudo chown -R www-data:www-data /opt/fuck0trust/server-node/data
```

## License

与主项目保持一致
