# 管理员后台

路径：`apps/admin`。Vue 3 + Vite + TypeScript + **antdv-next 1.5.x**。只给管理员用。不是第二套车主 App。开发用 Vite `:5173`（`/login`）；生产 `dist` 交给 Go（`FENGHUOLUN_ADMIN_DIR`），挂在 **`/admin/`**（`/admin/login`）。车主 H5 占 `/`。

## 谁能进

- 仅管理员会话
- 部署时用 `FENGHUOLUN_ADMIN_BOOTSTRAP_*` 创建第一位管理员
- 无「用车主 refresh_token 登录后台」
- 无公开注册

## 页面

| 路由 | 职责 |
|---|---|
| `/login`（生产 `/admin/login`） | 用户名密码 |
| `/`（生产 `/admin/`） | 总览：绑定数、今日同步成功/失败、凭证失效、上游连续失败 |
| `/bindings` | 绑定表：车型、脱敏 VIN、状态、最近同步；停用 / 恢复 / 立即同步 |
| `/bindings/:id` | 详情：脱敏元数据、凭证有/无、最近快照字段、任务；无凭证明文、无完整官方响应 |
| `/jobs` | 同步任务历史（追加日志，不是一车一行） |
| `/data` | 只读脱敏库表浏览，无 SQL、无导出 |
| `/settings` | 运行时设置：cron、CORS、快照保留、陈旧阈值；系统版本与一键更新 |
| `/account` | 改密码、退出 |

同步时间直接展示后端给的北京时间字符串，不要在浏览器里换时区。

停用绑定：只清本服务车主会话，不向官方下发车辆命令。立即同步走和 cron 同一条只读路径。

不做：远程车控面板、地图、车主模仿界面、编辑官方 token、SQL 控制台。

## 权限

前端路由守卫只是体验。所有敏感 GET/POST/PUT 以服务端管理员中间件为准。车主 token 访问 admin API 必须 403。

## 展示脱敏

- VIN：保留策略见 `10-security.md`（例如只显示后 4 位）
- 错误信息：展示 `errorInternal` 前仍要过滤 Bearer、cookie、token 字段
- 禁止下载 HAR、禁止导出凭证明文
- 库表浏览：密文列只显示有/无，会话不返回 token 哈希

## 运行时设置 vs 环境变量

| 在后台改（`app_settings`） | 只在 `.env` |
|---|---|
| 定时同步间隔（热替换 gcron） | `HTTP_ADDR` |
| CORS 来源 | `SQLITE_PATH` |
| 每车快照保留条数 | `TOKEN_KEK` |
| 快照陈旧阈值（秒） | 引导管理员账号（仅首次） |
| 管理员密码 | `ADMIN_DIR`（托管 dist，可选） |

环境变量是缺省：库空时写入 `app_settings`。

## UI

- 组件库：`antdv-next`，按组件 import，locale `zh_CN`
- 文案简体中文
- 不要 `antdv init`、不要 gf-vue-admin
- 官方 skill 在 `apps/admin/.cursor/skills/antdv-next/`（en-US 文档）；产品约束见 `apps/admin/AGENTS.md`

## 完成标准（运维台）

- [x] 登录后侧栏：总览 / 绑定 / 任务 / 数据 / 设置 / 账号
- [x] 总览数字可点到对应列表过滤
- [x] 绑定可停用、恢复、立即同步、进详情
- [x] 详情含快照主要字段（电量/续航/胎压/增程/12V/门窗）与能耗、最近任务
- [x] 任务历史含 `running` / 成功 / 失败；失败可看脱敏 `errorInternal`
- [x] 数据页时间为北京时间墙钟，无密文列、无完整 VIN
- [x] 非法 cron 不留下错误配置；改密作废其它管理员会话
- [x] 设置页可检查 GitHub Release；容器且挂了 docker.sock 时可一点更新到最新 `v*` 镜像

未做（有意）：车控、地图、SQL、导出、完整 VIN、token 明文、多管理员账号体系。
