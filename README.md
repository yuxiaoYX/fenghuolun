# 风火轮

面向 **哪吒 L** 车主的开源工具箱。用官方 App 的云接口，把车况和能耗看清楚，并把历史留在自己的服务里。

- 界面大字：**风火轮**
- 副题：哪吒 L 车况账本 · 车友自制
- 仓库：`fenghuolun`
- 哪吒 L（EP32）是第一个适配器，不是产品名
- 许可：MIT

第 **1** 期只读协议、第 **2** 期落库与管理员运维台已落地。权威说明只以 `docs/` 为准。下一期是车主前台蒸汽模式真机（第 3 期），不要做车控。

## 形态

| 端 | 路径 | 技术 | 谁用 |
|---|---|---|---|
| 车主前台 | [`apps/owner`](apps/owner) | uni-app x **蒸汽模式**（`uni-app-x.vapor`，不是 Steam 皮肤） | 仅车主 |
| 管理员后台 | [`apps/admin`](apps/admin) | Vue 3 + Vite + TypeScript + **antdv-next 1.5.x** | 仅管理员 |
| 后端 | [`server`](server) | GoFrame v2 + SQLite | 换票、拉数、落库、鉴权 |

车主填写官方 `refresh_token`，不开发官方短信登录。浏览器和 App **不直连**官方云。

**做不到也不宣传：** 替代官方 OTA / 给车机刷系统。

## 文档

1. [`docs/HANDOFF.md`](docs/HANDOFF.md)
2. [`docs/02-decisions.md`](docs/02-decisions.md)
3. [`docs/README.md`](docs/README.md)
4. [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) — 1Panel / Docker 生产部署

抓包附录只留本机 `docs/HAR-ANALYSIS.md`（已 gitignore，不进仓库）。

## 怎么跑

后端（在 `server/` 下会读本目录 `.env`，已 gitignore；进程环境变量优先）：

```bash
go test -C server ./...
go run -C server .
```

`GET http://127.0.0.1:8088/healthz` 应返回 `phase=2`。

复制 `server/.env.example` 为 `server/.env`，填写至少 32 字节的 `FENGHUOLUN_TOKEN_KEK`，重启后端。在车主「我的 → 绑定 / 更换 refresh_token」粘贴官方令牌。不要把 `.env` 或 token 提交进 git。默认不启用续航/电压 `/10`；要对候选缩放再设 `FENGHUOLUN_NETA_SCALE=candidate`，并对照同一时刻官方 App。不向车辆下发命令。

定时同步、CORS、快照保留、陈旧阈值：环境变量只作空库缺省，之后在管理端「设置」改（热替换 cron）。KEK、库路径、监听地址仍只在 `.env`。

管理员后台：

```bash
pnpm --dir apps/admin install
pnpm --dir apps/admin dev
```

浏览器打开终端里的 `http://127.0.0.1:5173`。先起后端。登录后是总览 / 绑定 / 任务 / 数据 / 设置 / 账号，不是一张总览表。部署时可 `pnpm --dir apps/admin build`，再设 `FENGHUOLUN_ADMIN_DIR` 指向 `apps/admin/dist`，由 Go 同端口托管（`/login` `/bindings` 等）。

车主前台：用 **HBuilderX 5.21+** 打开 `apps/owner`，确认蒸汽模式已勾选（`manifest.json` 里 `uni-app-x.vapor: true`）。运行与打包见 [`apps/owner/README.md`](apps/owner/README.md)。

`*.har` 已被 gitignore。本地 `1.har` 含账号与钥匙材料，禁止提交。

## 目录

```
apps/owner     车主 uni-app x（蒸汽模式）
apps/admin     管理员 Web（antdv-next）
server         Go
testdata/neta  解码单测用的脱敏官方响应样例
docs           权威文档
```

下一期见 [`docs/04-roadmap.md`](docs/04-roadmap.md) 第 3 期。不要做车控。
