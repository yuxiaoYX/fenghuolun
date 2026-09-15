# 新对话接手

把本文件和 `docs/02-decisions.md` 整份交给下一轮。不要依赖聊天记忆。根目录旧 Vite 原型已删除，不要再找 `src/`。

## 当前状态（2026-09-15）

- 产品方案与三端架构已锁定，见 `docs/`
- **第 1 期只读协议已通**（探活换票）。**第 2 期落库 + 管理员运维台已落地**
- 车主前台视觉/IA 已按 `docs/wireframes/hifi-v2.html` 改（五 Tab、座舱夜航）。车控只占位，不发写命令
- 后端是 `gf init` 结构（`server/main.go`、`api/`、`controller/`）。对外信封仍是 `{ok,data,error}`
- 官方 GoFrame Skills 在 `server/.cursor/skills/goframe-v2/`；antdv-next skill 在 `apps/admin/.cursor/skills/antdv-next/`（en-US）
- 无运行时假数据。绑定必须走官方 `refresh_token`。`server/.env` 填 `FENGHUOLUN_TOKEN_KEK` 后重启。healthz `phase=2`
- SQLite：`./data/fenghuolun.db`，经 gdb（contrib sqlite，无 cgo）；refresh/access/VIN 加密；会话只存哈希。行级 created/updated/deleted_at 由 ORM 写；解绑会话软删
- 管理员：antdv-next 运维台（总览 / 绑定 / 详情 / 任务 / 数据 / 设置 / 账号）。`sync_job` 追加历史（`manual|cron|admin`，含 `running`）。运行时设置在 `app_settings`，环境变量作空库缺省并热替换 gcron。部署可把 `apps/admin/dist` 交给 `FENGHUOLUN_ADMIN_DIR` 由 Go 托管
- KEK / SQLite 路径 / 监听地址仍只在 `.env`
- 快照 `stale` 阈值读 `app_settings.stale_after_sec`（默认 7200 秒），不是写死 2 小时
- 不要跳去车控，不要编造 sign 算法
- 胎压/胎温/内外温/里程/综合续航已对照官方 App「车辆详情」
- 门/锁/窗关闭侧已对照（0 / 16）；开侧未见，充电仍 unknown

## 必读顺序

1. [02-decisions.md](02-decisions.md)
2. [01-product.md](01-product.md)
3. [03-architecture.md](03-architecture.md)
4. [04-roadmap.md](04-roadmap.md)
5. 实现哪一层读哪一份：`06-backend` / `07-owner-app` / `08-admin` / `09-protocol-neta` / `10-security`

## 方案摘要（防遗忘）

- 哪吒 L 车主开源工具箱；产品名「风火轮」；仓库 `fenghuolun`
- 车主只用 uni-app x **蒸汽模式**前台（`manifest.json` → `uni-app-x.vapor: true`，不是 Steam 皮肤）
- 管理员只用 Vue3 Web + antdv-next 1.5.x；不要 Naive / Element / gf-vue-admin / `antdv init`
- Go **GoFrame v2** 后端；SQLite；浏览器不直连官方
- 车主填 `refresh_token`，不开发官方短信登录、不开发车主注册页
- 本服务仍要车主会话 + 管理员会话 + 车辆归属校验
- 不做 OTA、不做数字钥匙、未有闭环前不做车控
- 未知值保持未知；`fetchedAt` ≠ `reportedAt`；对外时间是北京时间墙钟（D22），前台不换算；能耗独立建模；位置仅存最新一个点（D17 修订，不存轨迹）
- `*.har` 不出库

## 明确不要做的「好心」

- 把蒸汽模式理解成 Steam UI 并改视觉
- 把车况雷达接口当官方接口
- 为了先跑通而编造换票 URL 或签名算法并当成已完成
- 在前台保存官方 Access Token
- 用本机 `now` 当车况时间
- 承诺替代官方 OTA
- 把管理端做成第二套车主座舱 / 地图 / SQL 控制台

## 下一轮默认任务

车主已有快照历史（`/snapshots`）和备注名。再往下默认：

1. 本机 HBuilderX 真机确认蒸汽模式（第 3 期唯一未勾项）
2. 对照官方 App：V3 电流、V4 开侧、V8 油箱
3. 可选：本机电价记账、车主导出 JSON

仍禁止车控、数字钥匙、编造 sign。
