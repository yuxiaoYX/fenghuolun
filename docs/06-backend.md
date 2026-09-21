# 后端（GoFrame v2）

模块路径建议：`github.com/yuxiaoYX/fenghuolun`（发布前可改；未发布时用仓内 `module fenghuolun`）。

框架：`github.com/gogf/gf/v2`。工程按 `gf init` 脚手架：`server/main.go` + `internal/cmd`。一个进程同时服务：

- `GET /healthz`（`status` / `phase` / `version`）
- `/api/v1/owner/*` 车主
- `/api/v1/admin/*` 管理员
- 可选托管 `apps/admin` 构建产物（`FENGHUOLUN_ADMIN_DIR` 指向 `dist`，同端口提供 `/` `/login` `/bindings` 等）

---

## 配置（环境变量）

| 变量 | 说明 |
|---|---|
| `FENGHUOLUN_HTTP_ADDR` | 如 `:8088` |
| `FENGHUOLUN_SQLITE_PATH` | 如 `./data/fenghuolun.db` |
| `FENGHUOLUN_TOKEN_KEK` | 凭证加密主密钥，至少 32 字节熵，**必填**（缺则进程起不来）。生产镜像入口会在未设置时写入数据目录 `token.kek` |
| `FENGHUOLUN_ADMIN_BOOTSTRAP_USER` | 仅当库中无管理员时创建 |
| `FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD` | 同上 |
| `FENGHUOLUN_CORS_ORIGINS` | 空库写入 `app_settings` 的缺省；之后以后台设置为准 |
| `FENGHUOLUN_CRON_SYNC` | 同上。默认关；如 `15m`。改后台后热替换 gcron，不必重启 |
| `FENGHUOLUN_NETA_SCALE` | 默认关（续航/电压为 —）。`candidate` 启用 `/10` 并带 decodeWarnings |
| `FENGHUOLUN_ADMIN_DIR` | 可选。管理端 `pnpm --dir apps/admin build` 的 `dist` 目录。空则不托管，开发用 Vite `:5173` |
| `FENGHUOLUN_UPDATE_REPO` | 查 GitHub Release 的 `owner/repo`，默认 `yuxiaoYX/fenghuolun`。设为 `-` 关闭检查 |
| `FENGHUOLUN_UPDATE_IMAGE` | 拉取的镜像名，默认 `ghcr.io/yuxiaoyx/fenghuolun` |
| `FENGHUOLUN_CONTAINER_NAME` | 本容器名，默认 `fenghuolun` |
| `FENGHUOLUN_UPDATE_DISABLE` | `1` 时后台「一点更新」关闭（仍可看版本） |

`HTTP_ADDR` / `SQLITE_PATH` / `TOKEN_KEK` / `ADMIN_DIR` 只在环境变量，后台改不了。`CORS` / `CRON` / 快照保留 / 陈旧阈值在 `app_settings`。会话是随机 ID + 哈希，没有 `SESSION_SECRET`。

启动时读取 `server/.env`（不覆盖已有进程环境）。密钥只写 `.env`，不要写 `manifest/config/config.yaml`。

禁止把官方 appKey、私钥、客户端证书写进默认配置文件并提交。若第 1 期验证必须客户端证书，用本地路径环境变量，文档只写变量名。

环境变量优先于 `manifest/config/config.yaml`。不要把密钥写进 yaml 并提交。

---

## 框架用法

- 路由：`api/*/v1` 的 `g.Meta` + `internal/controller`，在 `internal/boot` 里 `group.Bind`
- 开发：在 `server/` 下 `gf run main.go` / `gf gen ctrl`
- AI：官方 skill 在 `server/.cursor/skills/goframe-v2/`（[GoFrame Skills](https://goframe.org/ai/goframe-skills)）；产品信封与 neta 隔离见 `server/AGENTS.md`
- 对外 JSON 仍是本项目信封 `{ok,data,error}`，不用 gf 默认 `{code,message,data}`
- 管理员登录、车主 bind：写在 controller + `internal/owner`，**不要**引入 gf-vue-admin
- 落库：`gdb` + `github.com/gogf/gf/contrib/drivers/sqlite/v2`（pure go）。store 用 `gdb.New`（测试隔离，不走 `g.DB` 单例）+ `internal/model/do`。行时间交给 ORM，不要在 Data 里填 `created_at` / `updated_at` / `deleted_at`
- 定时同步：第 2 期 `gcron`
- `internal/neta` 不依赖 `ghttp`，保持可单测的纯解码 + HTTP 客户端

---

## HTTP 合同（本服务，不是官方）

统一信封：

```json
{ "ok": true, "data": {} }
{ "ok": false, "error": { "code": "token_invalid", "message": "请重新填写 refresh_token" } }
```

时间字段（`fetchedAt` / `reportedAt` / `syncedAt` / `fromAt` / `toAt`）是北京时间字符串 `YYYY-MM-DD HH:mm:ss`，零值为 `null`。不要输出 RFC3339 的 `Z`。实现：`internal/clock.Instant`。

错误码（稳定，前台依 code 分支）：

| code | 含义 |
|---|---|
| `unauthorized` | 无本服务会话 |
| `forbidden` | 角色不对或车辆不属于你 |
| `token_invalid` | 官方 refresh 失效 |
| `upstream` | 官方云失败 |
| `decode` | 响应结构变了 |
| `invalid_request` | 参数 |

### 车主

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/v1/owner/bind` | body: `{ "refresh_token": "..." }`，成功返回本服务会话 |
| `POST` | `/api/v1/owner/rebind` | 重新填写 refresh_token |
| `POST` | `/api/v1/owner/unbind` | 删除凭证与会话，快照是否保留可配置，默认保留历史、删凭证 |
| `GET` | `/api/v1/owner/vehicle` | 当前绑定摘要 |
| `PUT` | `/api/v1/owner/vehicle` | `{ "nickname" }`，最多 32 字；同步不覆盖车主备注 |
| `GET` | `/api/v1/owner/snapshot/latest` | 最新解码快照；另带计算字段 `stale`（上报超过 `stale_after_sec`，默认 7200） |
| `GET` | `/api/v1/owner/snapshots` | 本服务快照摘要分页；无完整 VIN |
| `GET` | `/api/v1/owner/energy?type=1` | 官方能耗、油量差分、充能记录、实付合计、容量反推 |
| `GET` | `/api/v1/owner/fills` | 充电/加油列表 |
| `POST` | `/api/v1/owner/fills` | 手补一条 |
| `GET/PUT/DELETE` | `/api/v1/owner/fills/{id}` | 看 / 补实付 / 删 |
| `POST` | `/api/v1/owner/sync` | 手动同步 |
| `GET` | `/api/v1/owner/sync/latest` | 最近一次任务状态 |

车主请求头：`Authorization: Bearer <owner_session>`。  
`bind` 例外：无会话。

### 管理员

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/v1/admin/login` | `{ "username", "password" }` |
| `POST` | `/api/v1/admin/logout` | 作废当前管理员会话 |
| `GET` | `/api/v1/admin/account` | `{ "username" }` |
| `POST` | `/api/v1/admin/password` | `{ "oldPassword", "newPassword" }`，新密码至少 8 位 |
| `GET` | `/api/v1/admin/health` | 绑定数、今日成功/失败、凭证失效、上游连续失败 |
| `GET` | `/api/v1/admin/bindings` | 列表：状态、车型、脱敏 VIN、最近同步 |
| `GET` | `/api/v1/admin/bindings/:id` | 详情，无凭证明文、无完整 VIN |
| `POST` | `/api/v1/admin/bindings/:id/disable` | 停用（清车主会话，不下发车控） |
| `POST` | `/api/v1/admin/bindings/:id/enable` | 恢复 |
| `POST` | `/api/v1/admin/bindings/:id/sync` | 管理员触发只读同步，`kind=admin` |
| `POST` | `/api/v1/admin/bindings/:id/kick` | 作废该车全部车主会话 |
| `GET` | `/api/v1/admin/jobs` | 任务历史；query：`bindingId` `status` `page` `pageSize` |
| `GET` | `/api/v1/admin/tables` | 可浏览表清单 |
| `GET` | `/api/v1/admin/tables/:name` | 脱敏分页；密文/哈希不返回 |
| `GET` | `/api/v1/admin/settings` | 运行时设置 |
| `PUT` | `/api/v1/admin/settings` | 写入 `app_settings` 并热替换 cron |
| `GET` | `/api/v1/admin/system` | 当前版本、最新 Release、是否可在后台更新。`?refresh=1` 跳过缓存 |
| `POST` | `/api/v1/admin/system/update` | 备份 SQLite、拉最新 `v*` 镜像、用 Docker 套接字重建本容器 |

管理员请求头与车主不同 audience，中间件分开。`sync_job` 是追加历史，不是一车一行。

---

## `internal/neta` 规则

1. **先证据后代码。** 路径、方法、Content-Type 以 HAR 已成功条目为准。
2. 已证实（只读，见 `09-protocol-neta.md`）：
   - `POST /pivot/mds-api/vehicleAccount/1.0/getCurrentVehicle`
   - `POST /pivot/veh-status/vehicle-status-control/1.0/getAppVehicleData`
   - 能耗两接口在 `https://api.chehezhi.cn`
3. 换票：`refreshApiToken` 已探活（V1）。`ErrRefreshUnverified` 仅作历史类型；禁止再猜其它换票 URL。
4. 官方业务码：主接口样本为 `code == 20000` 且数据在 `data`；数字钥匙是另一套信封，本期不接。
5. 请求头：HAR 见 `appId, appKey, timestamp, nonce, sign, Authorization` 等。未证明必要性之前，实现要可配置；**sign 算法未验证则标阻塞，禁止臆造 HMAC 糊弄联调。**
6. 解码与 HTTP 分文件：`client.go` / `decode_vehicle.go` / `decode_energy.go`。
7. 单测只跑 `testdata/neta/*.json` 脱敏样例。HTTP 测试注入假上游，禁止把假数据当产品绑定路径。

官方主机（HAR 所见，不是「保证长期有效」）：

| 主机 | 用途 |
|---|---|
| `appapi-pki.chehezhi.cn:18443` | 登录、车辆、车况 |
| `api.chehezhi.cn` | 能耗 |
| `certapi-pki.chehezhi.cn:18444` | 证书，样本 400 缺客户端证书 |
| `h5-battery.chehezhi.cn` | 官方 H5，浏览器 CORS 只允许它，与本服务无关 |

---

## 同步器

- 同一 `bindingId` 同时只允许一个 `running` job；再触发返回 `invalid_request`
- access 将过期则先换票
- 换票返回新 refresh 则原子覆盖
- 超时、5xx 记 `upstream`；官方鉴权失败记 `auth_failed` 并 `status=token_invalid`
- 不重试写命令（本期无写命令）

---

## 测试

- `decode_*_test.go`：脱敏样例 → 领域对象
- `http`：GoFrame named `g.Server` + HTTP 客户端，或等价黑盒测；假 neta client
- 禁止集成测试读取仓库内 `*.har` 或真实 token
