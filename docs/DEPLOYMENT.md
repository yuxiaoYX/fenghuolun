# 生产部署

本项目的生产容器包含 Go 后端和管理员前端。SQLite 数据库挂载到宿主机；官方车辆凭证的加密主密钥只通过环境变量注入。

生产只吃 **GitHub Release**（`v*` 标签）。`master` / `main` 上的镜像给 CI 和预览，不要用在这台生产机上。

## 服务器目录

```text
/opt/fenghuolun/deploy   docker-compose.yml 与 .env.production
/opt/fenghuolun/data     SQLite 数据库
/opt/fenghuolun/backups  备份
```

不必把整个 git 仓库放到运行目录。

## 1Panel / Compose

1. 将 `docker-compose.yml` 放到 `/opt/fenghuolun/deploy`。
2. 复制 `.env.production.example` 为 `.env.production`。
3. 填写 `FENGHUOLUN_TOKEN_KEK` 和正式 HTTPS 域名的 `FENGHUOLUN_CORS_ORIGINS`。
4. 首次空库启动时可临时填写管理员引导账号和密码；登录并修改密码后删除这两个变量，再重建容器。
5. 在 1Panel **容器 → 编排** 用「路径选择」导入 `/opt/fenghuolun/deploy` 并启动。不要设 `IMAGE_TAG`，默认就是 `latest`。

Compose 只把容器端口绑定到本机：`127.0.0.1:18088`。外部流量由 1Panel/OpenResty 反向代理到该端口。不要把 `18088` 或 `8088` 映射到公网。

第一次 `docker compose pull` 之前，仓库里必须已经有至少一个 `v*` 标签构建成功，否则 `latest` 还不存在。

命令行等价操作：

```bash
cd /opt/fenghuolun/deploy
docker compose pull
docker compose up -d --remove-orphans
curl http://127.0.0.1:18088/healthz
```

应返回含 `phase=2` 的响应。

## HTTPS

当前默认（域名和机器都在腾讯云）：

- DNS 的 A 记录指向服务器公网 IP。
- 1Panel 创建 **反向代理** 网站，目标 `http://127.0.0.1:18088`。
- 证书用 1Panel Let’s Encrypt（HTTP 验证），打开「HTTP 跳转到 HTTPS」。
- 腾讯云安全组与 1Panel 防火墙放行 `80` / `443`；不要开放 `8088` 或 `18088`。

以后若把源站换到香港并启用 Cloudflare：DNS 交给 Cloudflare，橙云代理，源站用 Origin 证书，SSL/TLS 选 `Full (strict)`。80/443 最好只允许 Cloudflare IP 段。不要用 Flexible。

## GitHub Actions 与 GHCR

`.github/workflows/container.yml` 会在 Pull Request 上测试并构建镜像。推送后发布到：

```text
ghcr.io/yuxiaoyx/fenghuolun
```

| 事件 | 镜像标签 | 谁用 |
|---|---|---|
| 推 `main` / `master` | 分支名、`sha-<短 SHA>` | CI / 预览 |
| 推 `v*` 标签 | `v0.1.0`、`sha-…`、**`latest`** | 生产 |

`latest` 只在 `v*` 标签构建时移动，默认分支推送不会改它。

工作流使用仓库自带的 `GITHUB_TOKEN`，不需要额外配置 Docker Hub 密钥。

如果 GHCR 包是私有的，在 1Panel **容器 → 仓库** 添加 `ghcr.io`，或在服务器上用只有 `read:packages` 权限的 GitHub PAT 登录一次：

```bash
echo "$GHCR_READ_TOKEN" | docker login ghcr.io -u yuxiaoYX --password-stdin
```

不要把 PAT 写入仓库或 `.env.production`。

## 发版

`master` 随时合。要上生产时打标签（不要用带连字符的预发布名，除非你有意让它成为 `latest`）：

```bash
git tag v0.1.0
git push origin v0.1.0
```

需要说明时再 `gh release create v0.1.0 --notes "…"`。等 Actions 把 `latest` 推上去之后，生产机才会拉到这一版。

## 更新与回滚

1Panel **计划任务** 用 Shell 定时拉 `latest`（合 PR 不会触发；发版后可「立即执行」）：

```bash
set -euo pipefail
cd /opt/fenghuolun/deploy
mkdir -p /opt/fenghuolun/backups
ts=$(date +%Y%m%d%H%M%S)
cp -a /opt/fenghuolun/data "/opt/fenghuolun/backups/pre-update-$ts"
docker compose pull
docker compose up -d --remove-orphans
curl -fsS http://127.0.0.1:18088/healthz | grep -q phase
```

另做每日目录备份：`/opt/fenghuolun/data` 和 `.env.production`，保留若干份。不要用 1Panel 缓存清理把旧镜像全删掉。

回滚时在编排环境变量里钉死上一版，再执行同样的 pull / up：

```text
IMAGE_TAG=v0.1.0
```

去掉 `IMAGE_TAG` 则回到跟随 `latest`。密钥和 SQLite 永远不放入镜像。
