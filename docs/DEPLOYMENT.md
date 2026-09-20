# 生产部署

本项目的生产容器包含 Go 后端和管理员前端。SQLite 数据库挂载到宿主机；官方车辆凭证的加密主密钥只通过环境变量注入。

## 服务器目录

```text
/opt/fenghuolun/deploy   仓库或部署文件
/opt/fenghuolun/data     SQLite 数据库
/opt/fenghuolun/backups  备份
```

## 1Panel / Compose

1. 将仓库放到 `/opt/fenghuolun/deploy`。
2. 复制 `.env.production.example` 为 `.env.production`。
3. 填写 `FENGHUOLUN_TOKEN_KEK` 和正式 HTTPS 域名的 `FENGHUOLUN_CORS_ORIGINS`。
4. 首次空库启动时可临时填写管理员引导账号和密码；登录并修改密码后删除这两个变量。
5. 在 1Panel 导入 `docker-compose.yml` 并启动。

Compose 只把容器端口绑定到本机：`127.0.0.1:18088`。外部流量应由 1Panel/OpenResty 反向代理到该端口。

命令行等价操作：

```bash
cd /opt/fenghuolun/deploy
docker compose pull
docker compose up -d --remove-orphans
curl http://127.0.0.1:18088/healthz
```

## HTTPS 与 Cloudflare

- DNS 的 A 记录指向服务器 IP，并打开 Cloudflare 橙色云朵。
- 1Panel 反向代理目标为 `http://127.0.0.1:18088`。
- Cloudflare SSL/TLS 使用 `Full (strict)`。
- 腾讯云安全组不要开放 `8088` 或 `18088`；80/443 最好只允许 Cloudflare IP 段。

## GitHub Actions 与 GHCR

`.github/workflows/container.yml` 会在 Pull Request 上测试并构建镜像，在 `main`/`master` 分支或 `v*` 标签推送时，将镜像发布到：

```text
ghcr.io/yuxiaoyx/fenghuolun
```

工作流使用仓库自带的 `GITHUB_TOKEN`，不需要额外配置 Docker Hub 密钥。

如果 GHCR 包是私有的，在服务器上创建一个只有 `read:packages` 权限的 GitHub PAT，然后登录一次：

```bash
echo "$GHCR_READ_TOKEN" | docker login ghcr.io -u yuxiaoYX --password-stdin
```

不要把 PAT 写入仓库或 `.env.production`。

## 更新与回滚

Compose 默认使用 `master` 镜像。生产环境建议把 `IMAGE_TAG` 固定为某个提交标签，例如 `sha-a1b2c3d`，在 1Panel 的 Compose 环境变量中设置：

```text
IMAGE_TAG=sha-a1b2c3d
```

发布新版本后，在 1Panel 中修改 `IMAGE_TAG`，然后重新部署；命令行等价操作：

```bash
cd /opt/fenghuolun/deploy
docker compose pull
docker compose up -d --remove-orphans
curl https://example.com/healthz
```

回滚时把 `IMAGE_TAG` 改回上一个正常版本，再执行同样的两条 Compose 命令。更新前先备份 `/opt/fenghuolun/data` 和 `.env.production`；密钥和 SQLite 永远不放入镜像。
