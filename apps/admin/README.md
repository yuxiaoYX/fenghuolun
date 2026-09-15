# apps/admin

管理员 Web。Vue 3 + Vite + TypeScript + **antdv-next 1.5.x**。只给管理员用。

```bash
cd apps/admin
pnpm install
pnpm dev
```

开发服务器 `http://127.0.0.1:5173`，把 `/healthz` 和 `/api` 代理到 Go `:8088`。

先起后端：`cd server && go run .`。

部署：`pnpm build` 得到 `dist/`，后端设 `FENGHUOLUN_ADMIN_DIR` 指向该目录，由 Go 同端口托管 `/` `/login` `/bindings` 等。`dist` 已 gitignore。

## AI Skill

官方 [Antdv Next Skills](https://github.com/antdv-next/skills) 已放入 `.cursor/skills/antdv-next/`。本仓库副本是官方发布的 **en-US** 文档（约 2026-07-03）；管理端界面文案仍用简体中文。产品约束见 [`AGENTS.md`](AGENTS.md)。

更新（在本目录）：

```bash
npx skills add antdv-next/skills -y --copy --agent cursor
```

装到 `./.cursor/skills/` 后覆盖即可。不要全局 `-g`。若要中文 references，在上游仓库跑 `pnpm run generate:zh` 再拷贝 `skills/antdv-next`。

本轮不接 `@antdv-next/cli` / MCP。
