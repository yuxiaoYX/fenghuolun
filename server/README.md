# server

GoFrame v2。用 `gf init` 的目录：`main.go`、`api/`、`internal/cmd|controller|boot`、`hack/`。

```bash
cd server
go test ./...
gf run main.go
# 或
go run .
```

默认 `:8088`（避开 HBuilderX 常用的 8080）。复制 `.env.example` 为 `.env`，填写 `FENGHUOLUN_TOKEN_KEK`。绑定写入 SQLite（默认 `./data/fenghuolun.db`），凭证 AES-GCM。

车主在前台粘贴官方 `refresh_token`。`POST /api/v1/owner/bind` 会换票再拉当前车辆、车况、能耗。没有运行时假数据模式。

```
GET  /healthz
POST /api/v1/owner/bind
GET  /api/v1/owner/vehicle
GET  /api/v1/owner/snapshot/latest
GET  /api/v1/owner/energy
POST /api/v1/owner/sync
```

`FENGHUOLUN_NETA_SCALE=candidate` 才启用续航/电压候选 `/10`。

脚手架命令（需已安装 `gf`）：

```bash
gf gen ctrl
gf gen dao
gf gen service
```

管理员：设置 `FENGHUOLUN_ADMIN_BOOTSTRAP_USER` / `PASSWORD` 后，Web 在 `apps/admin`（antdv-next）。开发用 Vite；部署 `pnpm --dir apps/admin build` 后设 `FENGHUOLUN_ADMIN_DIR` 指向 `dist`。合同见 `docs/06-backend.md`。不用 gf-vue-admin。

## AI Skill

官方 [GoFrame Skills](https://goframe.org/ai/goframe-skills) 已放在 `.cursor/skills/goframe-v2/`。约束见 `AGENTS.md`。
