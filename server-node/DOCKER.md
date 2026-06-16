# Docker 镜像自动构建配置

## GitHub Actions 自动构建

每次推送到 `main` 分支或修改 `server-node/` 目录时，会自动构建 Docker 镜像并推送到：

1. **GitHub Container Registry (ghcr.io)** - 免费，推荐
2. **Docker Hub** - 需要配置 Secrets

## 配置 Docker Hub（可选）

如果你想推送到 Docker Hub，需要设置以下 Secrets：

### 1. 创建 Docker Hub Access Token

1. 登录 [Docker Hub](https://hub.docker.com/)
2. 进入 Account Settings → Security → New Access Token
3. 创建一个新的 Access Token，权限选择 `Read, Write, Delete`
4. 复制生成的 Token（只会显示一次）

### 2. 在 GitHub 仓库设置 Secrets

进入你的 GitHub 仓库：

```
Settings → Secrets and variables → Actions → New repository secret
```

添加以下两个 Secrets：

| Secret 名称 | 值 |
|------------|---|
| `DOCKERHUB_USERNAME` | 你的 Docker Hub 用户名 |
| `DOCKERHUB_TOKEN` | 刚才创建的 Access Token |

### 3. 触发构建

推送代码即可自动触发构建：

```bash
git push
```

或手动触发：

```
GitHub → Actions → Build Docker Image → Run workflow
```

## 拉取镜像

### 从 GitHub Container Registry（推荐）

无需登录，直接拉取：

```bash
docker pull ghcr.io/patr1ck-shaw/fuck0trust-server:latest
```

### 从 Docker Hub

```bash
docker pull your-username/fuck0trust-server:latest
```

## 支持的平台

自动构建支持以下平台：
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM 架构，如树莓派)

## 镜像标签

| 标签 | 说明 |
|-----|------|
| `latest` | 最新的 main 分支构建 |
| `main` | main 分支最新构建 |
| `v1.0.0` | 特定版本（从 Git tag） |
| `sha-abc1234` | 特定 commit 的构建 |

## 本地多架构构建

如果需要本地构建多架构镜像：

```bash
# 创建 buildx builder
docker buildx create --name multiarch --use

# 构建并推送
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag your-username/fuck0trust-server:latest \
  --push \
  .
```

## 故障排查

### GitHub Actions 构建失败

1. 检查 Secrets 是否正确设置
2. 查看 Actions 日志获取详细错误
3. 如果不需要 Docker Hub，注释掉相关步骤

### 无法拉取 ghcr.io 镜像

GitHub Container Registry 默认是公开的，但如果是私有仓库：

```bash
# 使用 GitHub Personal Access Token 登录
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```
