# 部署指南

单进程：Go 后端 + 车主 H5 + 管理员前端。数据是一个 SQLite 文件。生产用 **GitHub Release**（`v*`）打出来的镜像；`master` / `main` 上的镜像给 CI 和预览。车主前台暂不编 App，浏览器打开即可。

```text
ghcr.io/yuxiaoyx/fenghuolun
```

1. [系统要求](#系统要求)
2. [Docker Compose（推荐）](#方式一docker-compose推荐)
3. [docker run](#方式二docker-run)
4. [一键脚本（1Panel / Linux）](#方式三一键脚本1panel--linux)
5. [源码构建](#方式四源码构建)
6. [配置说明](#配置说明)
7. [反向代理与 HTTPS](#反向代理与-https)
8. [升级](#升级)
9. [备份与恢复](#备份与恢复)
10. [回滚](#回滚)
11. [卸载](#卸载)
12. [排障](#排障)
13. [镜像与发版](#github-actions-与发版)

## 系统要求

- Docker 24+，Compose v2（`docker compose`）
- 64 位 Linux 最省心；macOS / Windows 用 Docker Desktop 同样能跑
- 默认端口 `8088`（可改）
- 磁盘：库文件很小，给数据目录留备份空间即可

不要把 `8088` 或 `18088` 直接映射到公网。生产请反代 + HTTPS，容器只听本机。

---

## 方式一：Docker Compose（推荐）

和常见开源项目一样：拉下来，起服务。

### 克隆仓库

```bash
git clone https://github.com/yuxiaoYX/fenghuolun.git
cd fenghuolun
bash deploy/init-env.sh          # Windows: pwsh -File deploy/init-env.ps1
docker compose up -d
```

`init-env` 会创建 `.env`（KEK + 首次管理员），并把密码打在屏幕上。车主打开 <http://127.0.0.1:8088/> ；管理员打开 <http://127.0.0.1:8088/admin/login> ，**立刻改密**。

健康检查：

```bash
curl -fsS http://127.0.0.1:8088/healthz
```

应看到 `phase=2`。

手动抄模板也可以：

```bash
cp .env.example .env
# 至少填写 FENGHUOLUN_TOKEN_KEK（openssl rand -base64 48）
# 以及首次 FENGHUOLUN_ADMIN_BOOTSTRAP_USER / PASSWORD
docker compose up -d
```

含自动生成密钥的镜像（本仓库 Dockerfile 的入口脚本）允许 `.env` 里 KEK 留空：容器会写 `data/token.kek`，重启复用。不要事后在 `.env` 里填一个**不同**的 KEK，否则旧绑定解不开。看自动密码：

```bash
cat data/admin-bootstrap.txt
docker compose logs fenghuolun
```

### 只下载编排文件（不克隆整仓）

```bash
mkdir -p fenghuolun && cd fenghuolun
curl -fsSL -O https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/docker-compose.yml
KEK=$(openssl rand -base64 48 | tr -d '\n')
PASS=$(openssl rand -hex 16)
printf 'FENGHUOLUN_TOKEN_KEK=%s\nFENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin\nFENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD=%s\n' "$KEK" "$PASS" > .env
echo "admin / $PASS"
docker compose up -d
```

这条路径只拉 GHCR 镜像，不要加 `--build`。把 `echo` 打出来的管理员密码存好。想改端口、CORS 或钉版本，再拉模板：

```bash
curl -fsSL -O https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/.env.example
# 把已生成的 KEK / 密码合并进 .env 后再 up
```

### 常用命令

```bash
docker compose ps
docker compose logs -f fenghuolun
docker compose pull && docker compose up -d
docker compose down          # 停止，保留 ./data
```

生产反代本机时，在 `.env` 写：

```bash
FENGHUOLUN_PUBLISH=127.0.0.1:18088:8088
```

然后 `docker compose up -d`。车主走 `https://你的域名/`，管理端走 `https://你的域名/admin/login`。

---

## 方式二：docker run

把密钥和首次密码换成你自己的随机串（不要每次重建都重新 `openssl rand`）：

```bash
export FENGHUOLUN_TOKEN_KEK="$(openssl rand -base64 48)"
export FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD="$(openssl rand -hex 16)"
echo "admin / $FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD"

docker run -d \
  --name fenghuolun \
  --restart unless-stopped \
  -p 8088:8088 \
  -v fenghuolun-data:/var/lib/fenghuolun \
  -e FENGHUOLUN_TOKEN_KEK \
  -e FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin \
  -e FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD \
  ghcr.io/yuxiaoyx/fenghuolun:latest
```

含入口脚本的镜像可以不传这两项：密钥会落到卷里的 `token.kek`，密码见 `docker logs` 或：

```bash
docker exec fenghuolun cat /var/lib/fenghuolun/admin-bootstrap.txt
```

已经备份过 `.env` 时：

```bash
docker run -d \
  --name fenghuolun \
  --restart unless-stopped \
  -p 127.0.0.1:18088:8088 \
  -v fenghuolun-data:/var/lib/fenghuolun \
  -e FENGHUOLUN_TOKEN_KEK='你的至少32字节随机串' \
  -e FENGHUOLUN_CORS_ORIGINS=https://你的域名 \
  -e FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin \
  -e FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD='首次密码' \
  ghcr.io/yuxiaoyx/fenghuolun:latest
```

镜像若是私有的：

```bash
docker login ghcr.io -u yuxiaoYX
# 密码用 GitHub PAT，权限只要 read:packages。不要把 PAT 写进 .env
```

---

## 方式三：一键脚本（1Panel / Linux）

适合已经装了 Docker 的服务器，不想把 git 仓库放在运行目录。容器只监听 **本机** `127.0.0.1:18088`。

```bash
curl -sSL https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/deploy/install.sh | bash -s -- --domain https://你的域名
```

没有域名可以先不加 `--domain`。脚本会：

1. 创建 `/opt/fenghuolun/{deploy,data,backups}`
2. 写入 `docker-compose.yml` 和 `.env.production`（含 KEK 与首次管理员密码）
3. 拉取 `ghcr.io/yuxiaoyx/fenghuolun:latest` 并启动
4. 检查 `http://127.0.0.1:18088/healthz`

密码打印在屏幕，并写入 `/opt/fenghuolun/deploy/admin-bootstrap.txt`。

接着在 1Panel 做域名和证书：

1. **网站 → 创建网站 → 反向代理**，目标 `http://127.0.0.1:18088`
2. 申请 Let’s Encrypt，打开「HTTP 跳转到 HTTPS」
3. 用脚本打印的账号登录 `https://你的域名/admin/login`，立刻改密
4. 从 `.env.production` 删除 `FENGHUOLUN_ADMIN_BOOTSTRAP_USER` / `PASSWORD` 两行，再执行 `/opt/fenghuolun/deploy/install.sh` 重建容器

镜像私有时，先在 **容器 → 仓库** 登录 `ghcr.io`（GitHub 用户名 + `read:packages` 的 PAT），再跑脚本。

升级（只在打了新的 `v*` 之后才会有新 `latest`）：

```bash
/opt/fenghuolun/deploy/install.sh upgrade
```

1Panel **计划任务** 把上面这一行设成每 6 小时即可。

回滚：

```bash
cd /opt/fenghuolun/deploy
IMAGE_TAG=v0.1.0 docker compose up -d
```

卸载（默认保留库和密钥）：

```bash
/opt/fenghuolun/deploy/install.sh uninstall
# 连数据一起删（不可恢复）：
# /opt/fenghuolun/deploy/install.sh uninstall --purge
```

### 服务器目录

```text
/opt/fenghuolun/deploy   docker-compose.yml、.env.production、install.sh
/opt/fenghuolun/data     SQLite 数据库、token.kek
/opt/fenghuolun/backups  升级前备份
```

不必把整个 git 仓库放到运行目录。

### 手动编排（1Panel「容器 → 编排」）

把仓库根目录的 `docker-compose.yml` 放到 `/opt/fenghuolun/deploy` 时，先改 `.env`：

```bash
FENGHUOLUN_PUBLISH=127.0.0.1:18088:8088
FENGHUOLUN_DATA_DIR=/opt/fenghuolun/data
```

也可以继续用脚本生成的那份 compose（写死了本机 `18088` 和 `/opt/fenghuolun/data`）。不要把 `18088` 或 `8088` 映射到公网。

---

## 方式四：源码构建

需要本机有 Docker（构建镜像）或 Go 1.23 + Node 22 + pnpm（直接跑）。

### 用仓库里的 Dockerfile

```bash
git clone https://github.com/yuxiaoYX/fenghuolun.git
cd fenghuolun
docker compose up -d --build
```

拉不下 GHCR 时也走这一条。本地 Dockerfile 带入口脚本，**可以不先写 `.env`**：密钥和首次密码会进 `./data/`。构建产物打成 compose 里的镜像名，随后 `docker compose up -d` 即可。

只构建、不走 compose：

```bash
docker build -t ghcr.io/yuxiaoyx/fenghuolun:local .
IMAGE_TAG=local docker compose up -d
```

### 不用 Docker（开发）

```bash
cp server/.env.example server/.env
# 填写至少 32 字节的 FENGHUOLUN_TOKEN_KEK
# 可选：FENGHUOLUN_ADMIN_BOOTSTRAP_USER / PASSWORD

go test -C server ./...
go run -C server .

pnpm --dir apps/admin install
pnpm --dir apps/admin dev
```

- API / healthz：<http://127.0.0.1:8088/healthz>
- 管理端开发：<http://127.0.0.1:5173> （Vite 把 `/api`、`/healthz` 代理到 Go）
- 部署本机托管管理端：`pnpm --dir apps/admin build`，再设 `FENGHUOLUN_ADMIN_DIR` 指向 `apps/admin/dist`
- 部署本机托管车主 H5：`pwsh -File apps/owner/build-h5.ps1`，再设 `FENGHUOLUN_OWNER_DIR` 指向 `apps/owner/h5-dist`

生产镜像已包含这两份静态文件。开发车主页：`pnpm owner:h5`。见 [`apps/owner/README.md`](../apps/owner/README.md)。

---

## 配置说明

容器内这三项由编排写死，一般不用改：

| 变量 | 容器内默认 |
|---|---|
| `FENGHUOLUN_HTTP_ADDR` | `0.0.0.0:8088` |
| `FENGHUOLUN_SQLITE_PATH` | `/var/lib/fenghuolun/fenghuolun.db` |
| `FENGHUOLUN_ADMIN_DIR` | `/app/admin` |
| `FENGHUOLUN_OWNER_DIR` | `/app/owner` |

你真正要动的是：

| 变量 | 说明 |
|---|---|
| `IMAGE_TAG` | 镜像标签，默认 `latest`（仅 `v*` Release 会移动） |
| `FENGHUOLUN_PUBLISH` | 端口映射，默认 `8088:8088` |
| `FENGHUOLUN_DATA_DIR` | 宿主机数据目录，默认 `./data` |
| `FENGHUOLUN_TOKEN_KEK` | 凭证加密主密钥；空则写入数据目录 `token.kek` |
| `FENGHUOLUN_CORS_ORIGINS` | 空库写入设置表的缺省；本机 localhost 始终放行 |
| `FENGHUOLUN_ADMIN_BOOTSTRAP_USER` | 仅当库中无管理员时创建 |
| `FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD` | 同上；空则写入 `admin-bootstrap.txt` |
| `FENGHUOLUN_CRON_SYNC` | 空库缺省。之后以后台「设置」为准并热替换 |
| `FENGHUOLUN_NETA_SCALE` | 默认关。`candidate` 才启用续航/电压 `/10` |

`HTTP_ADDR` / `SQLITE_PATH` / `TOKEN_KEK` / `ADMIN_DIR` 只在环境变量，后台改不了。登录后从 `.env` 删掉 `BOOTSTRAP_*` 两行，避免每次重建都把「首次密码」挂在环境里（已有管理员时这两项本来就是空操作）。

完整变量说明见 [`06-backend.md`](06-backend.md)。

---

## 反向代理与 HTTPS

当前默认（域名和机器都在腾讯云）：

- DNS 的 A 记录指向服务器公网 IP
- 反代目标 `http://127.0.0.1:8088`（Compose 默认）或 `http://127.0.0.1:18088`（`install.sh`）
- 证书用 1Panel Let’s Encrypt（HTTP 验证）或下面的 Caddy / Nginx
- 安全组与防火墙放行 `80` / `443`；不要开放 `8088` 或 `18088`

车主 H5 和管理端由 Go **同端口**托管：`/` 是车主，`/admin/` 是后台。反代到根路径即可，`/api`、`/healthz` 不用拆。

### 1Panel

**网站 → 创建网站 → 反向代理**，目标填容器在宿主机上的地址。申请证书，打开 HTTP 跳转 HTTPS。

### Caddy

```caddy
example.com {
    reverse_proxy 127.0.0.1:8088
}
```

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;

    # ssl_certificate     /path/to/fullchain.pem;
    # ssl_certificate_key /path/to/privkey.pem;

    client_max_body_size 16m;

    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

以后若把源站换到香港并启用 Cloudflare：DNS 交给 Cloudflare，橙云代理，源站用 Origin 证书，SSL/TLS 选 `Full (strict)`。80/443 最好只允许 Cloudflare IP 段。不要用 Flexible。

有公网域名时把 `FENGHUOLUN_CORS_ORIGINS=https://example.com` 写进 `.env`（或在后台「设置」里改）。同端口托管时通常不是硬需求，但填上更稳。

---

## 升级

### 管理后台一点更新（推荐）

不需要挂载 `/var/run/docker.sock`。容器要有 `restart: unless-stopped`（Compose 和 `docker run` 示例都有），数据卷要在。管理员登录 → **设置 → 系统更新 → 检查更新 / 更新到最新版**。会：

1. 把 SQLite 拷到数据目录 `backups/`
2. 从 GitHub Release 下载当前架构的程序包（二进制、管理端、车主页），并核对 `SHA256SUMS`
3. 写到数据卷 `app/current/`，进程退出。Docker 拉起同一个容器，入口改跑这一份

页面会断几秒到一两分钟。登录态还在。基础镜像里的系统包不会跟着换；那种情况仍用下面的命令行升级。

已经在跑、镜像里还没有这段逻辑的服务器，要先在宿主机执行一次 `docker compose pull && docker compose up -d`。从下一个带程序包的 Release 起，就可以在后台点。

某个 Release 还没上传程序包时，如果容器挂了 `docker.sock`，会退回旧办法：拉 GHCR 镜像并重建容器。不挂套接字就只显示版本，并提示用命令行。`docker-compose.yml` 里的套接字那一行可以删。

`FENGHUOLUN_UPDATE_DISABLE=1` 可关掉一点更新。

### 命令行

生产只跟 `v*` 的 `latest`（或把 `IMAGE_TAG` 钉死）。打 `v*` 标签后 Actions 会推 GHCR **并创建 GitHub Release**。git tag 和 GitHub Releases 页不是一回事：以前只推 tag 时，后台若只看 `/releases/latest` 会漏掉新版本；现在查找会同时看 tag，且发版工作流会把 Release 补上。

```bash
cd fenghuolun          # 或 /opt/fenghuolun/deploy
mkdir -p backups
cp -a data "backups/pre-update-$(date +%Y%m%d%H%M%S)"
docker compose pull
docker compose up -d
curl -fsS http://127.0.0.1:8088/healthz
```

用 `install.sh` 的机器直接：

```bash
/opt/fenghuolun/deploy/install.sh upgrade
```

克隆了仓库、想顺带更新编排文件：

```bash
git pull
docker compose pull
docker compose up -d
```

---

## 备份与恢复

必须**一起**备份：

- `fenghuolun.db`（以及 `-wal` / `-shm`，若存在）
- `token.kek`（若密钥不在 `.env` 里）
- `.env` / `.env.production`（若你把 KEK 写在这里）

Compose 默认数据在 `./data`：

```bash
docker compose stop
cp -a data "backups/fenghuolun-$(date +%Y%m%d)"
docker compose start
```

`docker run` 用了命名卷时：

```bash
docker run --rm -v fenghuolun-data:/data -v "$PWD/backups:/backup" debian:bookworm-slim \
  tar czf /backup/fenghuolun-data.tgz -C /data .
```

恢复：停容器，把目录盖回去，再启动。KEK 和库必须是同一对，否则凭证解不开，只能让车主重新绑定。

---

## 回滚

```bash
# 先有备份再换标签
IMAGE_TAG=v0.1.0 docker compose up -d
```

或在 `.env` 把 `IMAGE_TAG` 改成旧标签后 `docker compose up -d`。

镜像不存在时去 [GHCR](https://github.com/yuxiaoYX/fenghuolun/pkgs/container/fenghuolun) 看实际标签。

---

## 卸载

```bash
docker compose down
# 数据还在 ./data ；连数据一起删：
# rm -rf data
```

`docker run`：

```bash
docker rm -f fenghuolun
# docker volume rm fenghuolun-data
```

---

## 排障

| 现象 | 处理 |
|---|---|
| `healthz` 无响应 | `docker compose logs fenghuolun`。缺 KEK 时旧镜像会直接退出；现在的镜像会自动生成 |
| 拉不下 `ghcr.io/...` | `docker login ghcr.io`，或 `docker compose up -d --build` |
| 登录不了 | `cat data/admin-bootstrap.txt`；改密后不要再用这个文件里的旧密码 |
| 页面能开、接口 CORS 失败 | 把站点 `https://域名` 写入 CORS（`.env` 或后台设置） |
| 绑定后凭证解不开 | KEK 和这套库不是一对。不要换 `token.kek` / `.env` 里的 KEK |
| 端口占用 | `.env` 里改 `FENGHUOLUN_PUBLISH=8089:8088` |
| Windows 下 bash 脚本跑不了 | `pwsh -File deploy/init-env.ps1`，再 `docker compose up -d` |
| `latest` 不是你刚合的 master | 生产 `latest` 只在打 `v*` 标签时移动 |
| 设置里「更新到最新版」是灰的 | 不是容器、已是最新，或这个 Release 还没有程序包。看该卡片提示；或宿主机 `docker compose pull && docker compose up -d` |
| 点了更新一直转圈 | 看 `docker logs fenghuolun` 和数据目录 `update-last-error.json`。旧的镜像重建方式才有 `fenghuolun-updater` |

容器已启动但本机访问失败时，确认映射：

```bash
docker compose ps
docker port fenghuolun
```

---

## GitHub Actions 与发版

[`.github/workflows/container.yml`](../.github/workflows/container.yml) 在 Pull Request 上测试并构建镜像。推送后发布到 `ghcr.io/yuxiaoyx/fenghuolun`。

| 事件 | 镜像标签 | 谁用 |
|---|---|---|
| 推 `main` / `master` | 分支名、`sha-<短 SHA>` | CI / 预览 |
| 推 `v*` 标签 | `v0.1.0`、`sha-…`、**`latest`** | 生产 |

`latest` 只在 `v*` 标签构建时移动，默认分支推送不会改它。工作流使用仓库自带的 `GITHUB_TOKEN`。

`master` 随时合。要上生产时打标签（不要用带连字符的预发布名，除非你有意让它成为 `latest`）：

```bash
git tag v0.1.1
git push origin v0.1.1
```

等 Actions 把 `latest` 推上去之后，生产机 `docker compose pull && docker compose up -d`（或 `install.sh upgrade`）才会拉到这一版。
