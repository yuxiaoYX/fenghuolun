# apps/admin — 管理员 Web

本目录才是管理端工程根。不要把仓库根 `fenghuolun/` 当 Vite 工程跑。

## 官方 Skill

已放入 `.cursor/skills/antdv-next/`（来源：[antdv-next/skills](https://github.com/antdv-next/skills)）。写 Layout / Table / Form / Menu 等组件时先读该 skill 的 `SKILL.md` 与对应 `references/components/<name>/`。

更新（在本目录）：

```bash
npx skills add antdv-next/skills -y --copy --agent cursor
```

装到 `./.cursor/skills/` 后覆盖即可。不要全局 `-g` 替代本仓库副本。

官方发布的 skill 目前是 **en-US**（`GENERATION.md` 记 2026-07-03）。组件 API 以这份离线文档为准；**界面文案仍用简体中文**。不要把 demo 里的英文按钮字抄进产品。若要中文 references，需在上游仓库跑 `pnpm run generate:zh` 再拷贝 `skills/antdv-next`。

## 本工程覆盖官方 skill 的点

官方 skill 是通用组件库。本产品下列约定 **优先**：

- 只服务管理员；不是第二套车主 App，不要做成座舱夜航 / 俯视车图
- 只请求本仓库 Go 后端 `/api/v1/admin/*` 与 `/healthz`。禁止直连官方云、禁止展示凭证明文 / 完整 VIN / token
- 对外 JSON 是 `{ok,data,error}`，时间字段已是北京时间墙钟，前台不要再换时区
- 不做车控、不做地图、不开放 SQL 控制台
- 不要引入 gf-vue-admin；不要 `antdv init` 另起工程
- 车主前台在 `apps/owner`，不要把 `antdv-next` 装到那边

## CLI / MCP（这轮没接）

`@antdv-next/cli` 和 `antdv mcp` **没有**接入本工程。DeepSeek Harness 本会话也没有该 MCP。查 API 读本地 `references/`，或打开 [antdv-next 组件文档](https://www.antdv-next.com/components/overview-cn)。不要假装调过 CLI / MCP。

## 入口

- `pnpm --dir apps/admin dev` → `http://127.0.0.1:5173`
- 先起后端：`go run -C server .`（默认 `:8088`）
- 部署：`pnpm --dir apps/admin build`，`FENGHUOLUN_ADMIN_DIR` 指向 `dist`
