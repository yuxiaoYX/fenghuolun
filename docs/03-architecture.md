# 架构

## 目标结构（重构后）

当前根目录的 Vite 应用 **不是** 目标形态。目标单仓：

```
fenghuolun/
  README.md
  LICENSE
  docs/                 权威文档（本目录）
  apps/
    owner/              uni-app x 车主前台（蒸汽模式）
    admin/              Vue3 管理员 Web
  server/               GoFrame v2 模块（gf init 结构）
    main.go             gf cmd 入口
    api/                请求/响应 + g.Meta 路由
    hack/               gf CLI（gen ctrl/dao）
    manifest/config/
    internal/
      cmd/              gcmd.Command
      boot/             注册路由
      controller/       health / owner
      middleware/       信封、CORS、车主会话
      owner/            绑定与同步业务
      neta/             官方协议客户端与解码（不依赖 ghttp）
      store/            第 1 期内存；第 2 期 gdb+SQLite
      dao/ model/ service/  留给 gf gen
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
| admin | 列表、状态、脱敏日志、禁用某绑定 | 用管理员身份拉某一辆车的明文 token、代替车主发车控 |
| server | 换票、官方 HTTP、解码、加密存储、鉴权 | 把官方响应原样倒给前端（必须先解码/脱敏） |

## 数据流

### 绑定

1. 车主提交 `refresh_token`
2. `internal/neta` 换票（路径待验证，见 V1）
3. 调用 `getCurrentVehicle`，得到 VIN、车型、绑定关系
4. 用 KEK 加密 refresh/access 入库
5. 签发本服务 `owner_session`
6. 前台只存本服务会话

### 同步（定时或手动）

1. 取出凭证，必要时换票；若官方返回新 refresh，覆盖存储
2. `getAppVehicleData` → 解码为 `VehicleSnapshot`
3. `findEnergyConsumptionStatistics` + `queryEnergyConsumptionByVin` → `OfficialEnergy*`
4. 分别写入 `fetchedAt` / `reportedAt`
5. 失败写入 `sync_job`（错误分类：auth / upstream / decode），**日志不含 token 与完整 VIN**（VIN 只存脱敏或哈希+后四位策略见安全文档）

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

## 部署形态（一期）

- 单机自托管：一个 Go 进程 + 本地 SQLite 文件 + 静态托管 admin；owner 以 App 安装包或连接该 API 的 H5
- CORS：admin 与 owner H5 的来源白名单来自环境变量
- 不默认公网多租户 SaaS。若同一实例服务多个车主，仍然是「多个 refresh_token 绑定」，不是开放注册社区

## 与现网官方云的关系

本服务是 **官方云的客户端 + 本地历史库**。官方云不可用时：

- 同步失败，前台可读最近一次成功快照
- 不得假装实时

## 已删除的旧件（第 0 期）

- 根目录 `src/`、`vite.config.ts`、`vite-official-proxy.ts`、`index.html`、旧 tsconfig
- 浏览器指定 `X-Upstream` 的官方代理
- 根目录不再是 Vue 应用；`package.json` 只保留 workspace 脚本
