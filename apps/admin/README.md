# apps/admin

管理员 Web。Vue 3 + Vite。只给管理员用。

```bash
cd apps/admin
pnpm install
pnpm dev
```

开发服务器 `http://127.0.0.1:5173`，把 `/healthz` 和 `/api` 代理到 Go `:8088`。

先起后端：`cd server && go run .`。
