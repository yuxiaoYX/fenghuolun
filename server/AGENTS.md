# apps 不对，这里是 server — GoFrame v2

本目录才是 Go 模块根。不要把仓库根 `fenghuolun/` 当 gf 工程跑。

## 官方 Skill

已放入 `.cursor/skills/goframe-v2/`（来源：[GoFrame Skills](https://goframe.org/ai/goframe-skills)，仓库 [gogf/skills](https://github.com/gogf/skills)）。写 `api/`、`controller/`、`gf gen`、gdb/dao 时先读该 skill。

更新（在本目录）：

```bash
npx skills add github.com/gogf/skills
```

装到 `./.cursor/skills/` 后覆盖即可。不要全局 `-g` 替代本仓库副本。

## 本工程覆盖官方 skill 的点

官方 skill 是通用 GoFrame。本产品下列约定 **优先**：

- 对外 JSON 是 `{ok,data,error}`，不用 gf 默认 `{code,message,data}`
- 不要引入 gf-vue-admin
- `internal/neta` 不依赖 `ghttp`，禁止编造官方 `sign`
- 落库已是 `gdb` + contrib sqlite。测试用 `gdb.New` 隔离库，不走 `g.DB` 单例，因此未接 `gf gen dao`（CLI 默认也不带 sqlite 驱动）
- 业务在 `internal/owner`，不是再套一层 gf-vue 后台

## 入口

- `gf run main.go` 或 `go run .`
- 路由：`api/*/v1` 的 `g.Meta` + `internal/controller` + `internal/boot`
