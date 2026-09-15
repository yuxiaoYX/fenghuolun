# 架构

## 目标结构

根目录旧 Vite 原型已删除。当前单仓：

```
fenghuolun/
  README.md
  LICENSE
  docs/                 权威文档（本目录）
  apps/
    owner/              uni-app x 车主前台（蒸汽模式）
    admin/              Vue3 + antdv-next 管理员 Web
  server/               GoFrame v2 模块（gf init 结构）
    main.go             gf cmd 入口
    api/                请求/响应 + g.Meta 路由
    hack/               gf CLI（gen ctrl/dao）
    manifest/config/
    internal/
      cmd/              gcmd.Command
      boot/             注册路由
      controller/       health / owner / admin
      middleware/       信封、CORS、车主/管理员会话
      owner/            绑定与同步业务
      neta/             官方协议客户端与解码（不依赖 ghttp）
      store/            gdb + SQLite（contrib sqlite，无 cgo）
      model/do|entity   表结构
  testdata/
    neta/               仅解码单测用的脱敏 JSON
```

允许暂时没有 `packages/`。禁止再把官方客户端写进前台。

## 三端职责

```
车主 uni-app x          管理员 Vue3 Web
   │ 本服务会话                │ 管理员会话
   ▼                          ▼
              Go server
     ┌────────────┼────────────┐
     │ 归属校验   │ 加密凭证    │ 落库
     ▼            ▼            ▼
           neta adapter
     只在服务端访问官方云
  appapi-pki / api.chehezhi.cn / …
```

| 端 | 可以做 | 不可以做 |
|---|---|---|
| owner | 展示解码后的领域对象、提交 refresh_token、请求同步 | 持有官方 Access Token、自拼签名、直连官方 |
| admin | 绑定运维、任务历史、脱敏库表、运行时设置 | 用管理员身份拉明文 token、代替车主发车控、SQL 控制台、地图 |
| server | 换票、官方 HTTP、解码、加密存储、鉴权 | 把官方响应原样倒给前端（必须先解码/脱敏） |

## 数据流

### 绑定

1. 车主提交 `refresh_token`
2. `internal/neta` 换票（`refreshApiToken`，V1 已探活）
3. 调用 `getCurrentVehicle`，得到 VIN、车型、绑定关系
4. 用 KEK 加密 refresh/access 入库
5. 签发本服务 `owner_session`
6. 前台只存本服务会话

### 同步（定时、车主手动、管理员立即同步）

1. 取出凭证，必要时换票；若官方返回新 refresh，覆盖存储
2. `getAppVehicleData` → 解码为 `VehicleSnapshot`
3. `queryEnergyConsumptionByVin` → `OfficialEnergy*`
4. 分别写入 `fetchedAt` / `reportedAt`
5. `sync_job` 追加一行：先 `running`，结束写 `ok` / `auth_failed` / `upstream` / `decode`。**日志不含 token 与完整 VIN**

### 展示

- 前台只请求本服务 `/api/v1/owner/*`
- 后台只请求 `/api/v1/admin/*`
- 两套路由、两套中间件，禁止共用「有 token 就放行」

## 适配器边界

```
官方 JSON  ──decode──►  领域对象  ──JSON──►  前台
                ▲
         internal/neta
         单位、枚举、无效值、信封 code
```

领域对象是 UI 与后台列表的唯一合同。官方字段名不得泄漏到 uvue 页面里做临时解析。

哪个 source 由 `source_id = neta` 标识。以后的车型是新的 `internal/<brand>`，不是 if-else 洒在 HTTP 层。

## 部署形态

- 单机自托管：一个 Go 进程 + 本地 SQLite 文件；admin 用 Vite 开发代理或静态托管构建产物；owner 以 App 安装包或连接该 API 的 H5
- CORS：空库时从环境变量写入 `app_settings`，之后以后台设置为准；本机 `127.0.0.1` / `localhost` 端口始终放行
- 不默认公网多租户 SaaS。若同一实例服务多个车主，仍然是「多个 refresh_token 绑定」，不是开放注册社区

## 与现网官方云的关系

本服务是 **官方云的客户端 + 本地历史库**。官方云不可用时：

- 同步失败，前台可读最近一次成功快照
- 不得假装实时

## 已删除的旧件（第 0 期）

- 根目录 `src/`、`vite.config.ts`、`vite-official-proxy.ts`、`index.html`、旧 tsconfig
- 浏览器指定 `X-Upstream` 的官方代理
- 根目录不再是 Vue 应用
