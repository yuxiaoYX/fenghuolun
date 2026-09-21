# 生产部署

本项目的生产容器包含 Go 后端和管理员前端。SQLite 数据库挂载到宿主机；官方车辆凭证的加密主密钥只通过环境变量注入。

生产只吃 **GitHub Release**（`v*` 标签）。`master` / `main` 上的镜像给 CI 和预览，不要用在这台生产机上。

## 一键部署（推荐）

1Panel **终端**（或任意已装 Docker 的 Linux）执行：

```bash
curl -sSL https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/deploy/install.sh | bash -s -- --domain https://你的域名
```

没有域名可以先不加 `--domain`。脚本会：

1. 创建 `/opt/fenghuolun/{deploy,data,backups}`
2. 写入 `docker-compose.yml`
3. 生成 `FENGHUOLUN_TOKEN_KEK` 和首次管理员密码（密码打印在屏幕，并写入 `deploy/admin-bootstrap.txt`）
4. 拉取 `ghcr.io/yuxiaoyx/fenghuolun:latest` 并启动
5. 检查 `http://127.0.0.1:18088/healthz`

容器只监听本机 `127.0.0.1:18088`。接着在 1Panel 做域名和证书：

1. **网站 → 创建网站 → 反向代理**，目标 `http://127.0.0.1:18088`
2. 申请 Let’s Encrypt，打开「HTTP 跳转到 HTTPS」
3. 用脚本打印的账号登录 `https://你的域名/login`，立刻改密
4. 从 `.env.production` 删除 `FENGHUOLUN_ADMIN_BOOTSTRAP_USER` / `PASSWORD` 两行，再执行 `/opt/fenghuolun/deploy/install.sh` 重建容器

镜像若是私有的，先在 **容器 → 仓库** 登录 `ghcr.io`（GitHub 用户名 + `read:packages` 的 PAT），再跑脚本。不要把 PAT 写进 `.env.production`。

升级（只在打了新的 `v*` 之后才会有新 `latest`）：

```bash
/opt/fenghuolun/deploy/install.sh upgrade
```

1Panel **计划任务** 把上面这一行设成每 6 小时即可，不必手写 `docker compose`。

回滚：

```bash
cd /opt/fenghuolun/deploy
IMAGE_TAG=v0.1.0 docker compose up -d
```

卸载（默认保留库和密钥）：

```bash
/opt/fenghuolun/deploy/install.sh uninstall
```

## 服务器目录

```text
/opt/fenghuolun/deploy   docker-compose.yml、.env.production、install.sh
/opt/fenghuolun/data     SQLite 数据库
/opt/fenghuolun/backups  升级前备份
```

不必把整个 git 仓库放到运行目录。

## 手动编排

不走脚本时：把仓库根目录的 `docker-compose.yml` 放到 `/opt/fenghuolun/deploy`，复制 `.env.production.example` 为 `.env.production`，填写 KEK 与 CORS，1Panel **容器 → 编排** 用「路径选择」导入该目录。不要设 `IMAGE_TAG`，默认就是 `latest`。不要把 `18088` 或 `8088` 映射到公网。

## HTTPS

当前默认（域名和机器都在腾讯云）：

- DNS 的 A 记录指向服务器公网 IP。
- 1Panel 反向代理目标 `http://127.0.0.1:18088`。
- 证书用 1Panel Let’s Encrypt（HTTP 验证）。
- 安全组与 1Panel 防火墙放行 `80` / `443`；不要开放 `8088` 或 `18088`。

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

工作流使用仓库自带的 `GITHUB_TOKEN`。私有包登录方式见上文一键部署。

## 发版

`master` 随时合。要上生产时打标签（不要用带连字符的预发布名，除非你有意让它成为 `latest`）：

```bash
git tag v0.1.1
git push origin v0.1.1
```

等 Actions 把 `latest` 推上去之后，生产机执行 `install.sh upgrade` 才会拉到这一版。
