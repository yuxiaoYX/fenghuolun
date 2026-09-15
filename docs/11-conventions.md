# 约定

为避免后续开发把方案做丢，所有贡献（含 AI 续写）遵守下列规矩。

## 改代码前

1. 读 `docs/HANDOFF.md` 与 `docs/02-decisions.md`
2. 官方 path/字段以 `09-protocol-neta.md` 为准，无则标待验证
3. 与决策冲突的「优化」先改决策文件，再改代码

## 改文档

- 锁定项变更：必须写 `02-decisions.md` 变更记录
- 换票等协议实锤：更新 `09-protocol-neta.md`，必要时补本机 `HAR-ANALYSIS.md` 附录（脱敏且不进仓库）
- 分期完成：勾选 `04-roadmap.md` 的 checkbox
- 禁止只在聊天里达成新决策

## 代码

- Go：官方解码与 HTTP 层分开；错误类型能映射到 API `code`
- 时间：对外用 `clock.Instant`（北京时间墙钟）；不要在前台把 UTC 转本地
- 落库：用 do 对象；行级 `created_at` / `updated_at` / `deleted_at` 交给 gdb，禁止手写；软删用 `Delete()`
- 前端：不出现官方 host 常量（除文档）
- 测试夹具脱敏
- 未知不猜测：与其错误换算，不如返回 null
- 不要为了让 UI 好看把 `pluggedIn` 默认成 false

## 仓库边界

- 根目录不是 Vue / uni-app 工程；车主在 `apps/owner`，管理端在 `apps/admin`，后端在 `server/`
- 不要删除 `docs/`（本文档体系）、`LICENSE`、`.gitignore` 的 `*.har` 规则
- `1.har` 若仍在工作区：保持忽略，不要移动到 `testdata`

## 语言

- 用户界面：简体中文
- 代码标识符：英文
- 文档：简体中文

## 对 AI 续作者

若用户说「文档可能有问题」，是指 **本体系落地之前** 的旧 README / 旧 `src` 注释 / `obsolete/`。不是授权忽略 `docs/02-decisions.md`。
