# 数据模型

领域对象是前后端合同。官方原始 JSON 只留在 `internal/neta` 与加密/受限的 `raw` 列策略里。

所有可空业务量用 `null` 表示未知，不用 `0`、不用 `false` 冒充。

---

## 1. 领域对象

### 1.1 VehicleBinding 绑定

| 字段 | 说明 |
|---|---|
| `id` | 本服务主键 |
| `sourceId` | 目前恒为 `neta` |
| `vin` | 完整 VIN 只存在服务端；API 对外默认脱敏 |
| `officialVehicleId` | 官方车辆 id，**不是** VIN，也不是第三方 userId |
| `modelCode` | 如 `EP32` |
| `modelName` | 如 哪吒 L |
| `displayName` / `nickname` | 车主可改的备注名（`PUT /owner/vehicle`）；同步不覆盖 |
| `status` | `active` / `token_invalid` / `disabled` |
| `createdAt` / `updatedAt` | |

### 1.2 OfficialCredential 官方凭证（永不下发到任何前端）

| 字段 | 说明 |
|---|---|
| `bindingId` | |
| `refreshTokenEnc` | 加密后的 refresh_token |
| `accessTokenEnc` | 加密后的 access_token |
| `accessExpiresAt` | 官方 `expires_in` 折算；HAR 样本约 7 天，**不要写死** |
| `rotatedAt` | 最近一次换票 |

### 1.3 OwnerSession 车主会话

| 字段 | 说明 |
|---|---|
| `id` / `tokenHash` | 只存哈希 |
| `bindingId` | |
| `expiresAt` | |
| `createdAt` / `lastSeenAt` | |

绑定成功才有会话。失效 token 时会话可保留但业务接口返回「需重新绑定」。

### 1.4 AdminUser 管理员

| 字段 | 说明 |
|---|---|
| `id` | |
| `username` | |
| `passwordHash` | |
| `createdAt` | |

与车主无外键关系。禁止用同一套 JWT audience。

### 1.5 VehicleSnapshot 车况快照（解码后）

对应前台「车况」「电池」的只读合同。

| 字段 | 类型 | 规则 |
|---|---|---|
| `id` | string | |
| `bindingId` | string | |
| `sourceId` | `neta` | |
| `fetchedAt` | datetime | 本服务拉到官方数据的时刻（不是行 `created_at`） |
| `reportedAt` | datetime \| null | 车辆上报；没有则为 null，**禁止填 fetchedAt** |
| `stale` | bool | 仅快照 API 计算，不落库。`reportedAt` 距现在超过 `app_settings.stale_after_sec`（默认 7200）为 true；无 `reportedAt` 为 false |
| `online` | boolean \| null | 官方在线状态；未知为 null，UI 不写成离线 |
| `decodeWarning` | string[] | 如 `scale_unverified`、`sentinel_65535` |
| `vehicle` | object | `make, model, modelCode, name`；对外 API 默认无完整 VIN |
| `power.socPct` | number \| null | |
| `power.evRangeKm` | number \| null | `/10`，已对照综合续航 |
| `power.totalRangeKm` | number \| null | |
| `power.chargeStatus` | `unknown` \| `unplugged` \| `idle` \| `charging` \| `complete` | 默认 `unknown` |
| `power.pluggedIn` | boolean \| null | 未知必须 null，禁止默认 false |
| `power.chargePowerKw` | number \| null | |
| `extender` | object \| null | 非增程车型必须 `null`，UI 整卡隐藏 |
| `extender.fuelPct` | number \| null | |
| `extender.fuelRangeKm` | number \| null | |
| `extender.enabled` / `generating` | boolean \| null | |
| `body.locked` | `closed` \| `unknown` | 四门锁均为 0 才 closed |
| `body.doors.*` / `body.windows.*` | `closed` \| `unknown` | 门/锁 0、四窗 16、天窗 0；开侧未见 |
| `odometerKm` | number \| null | `mileage/10` |
| `climate.cabinC` / `outsideC` | number \| null | `(airIn/OutTemp-110)/2` |
| `battery12v.volts` | number \| null | `/10`，已证实（量纲唯一） |
| `tires.unit` | `bar` \| `unknown` | |
| `tires.*.bar` / `tempC` | number \| null | 压 `/55` 截 2 位；温 `-50` |
| `battery.packVoltageV` | number \| null | `/10`，已证实（冗余互验+量纲+国标惯例） |
| `battery.currentA` | number \| null | 强候选 (raw−10000)/10，**仍 null**，等三态路试 |
| `battery.cell*` / `bmsSohPct` / `chargeSohPct` | 一律允许 null | 无证据禁止计算 SOH |

**位置字段进入此对象。** `location.lng/lat` 从官方 `vehicleExtend.lng/lat`（÷1e6，WGS-84）解码，仅保留最新一个点；无有效坐标时为 null。决策 D17 已修订为采集并展示（车主+管理员可见），但仍不存历史轨迹。

### 1.6 OfficialEnergyStat 官方能耗（独立模型）

来源：`findEnergyConsumptionStatistics`、`queryEnergyConsumptionByVin`。

| 字段 | 说明 |
|---|---|
| `bindingId` | |
| `periodType` | `1` 周 / `2` 月 / `3` 年（官方 `type = 选项索引 + 1`；除 `1` 外待验证） |
| `countTime` | 官方日期分组，不要改成「含今天近七日」 |
| `totalKwh` | 对应 `totalConsumesEnergy`，UI 标注 kWh |
| `drivingKwh` | |
| `acKwh` | |
| `recoveryKwh` | 回收，单独展示，不要减进总量除非多样本证明 |
| `windKwh` | |
| `accessoryKwh` | |
| `fetchedAt` | |

**没有** `evRangeKm` / `fuelL` / `hybridRangeKm` 就不要出现在本对象。禁止为了旧 UI 填 0。油耗不是这个对象的字段。

### 1.6b ExtenderFuelLedger 增程油耗（本服务自建）

来源：连续 `VehicleSnapshot.extender.fuelPct`（及 `fuelRangeKm`），**不是** 官方能耗接口。

| 字段 | 说明 |
|---|---|
| `fromAt` / `toAt` | 两笔快照时间 |
| `fuelPctFrom` / `fuelPctTo` | |
| `fuelPctDelta` | 下降为耗油，上升为加油 |
| `kind` | `burn` / `refuel` / `flat` / `unknown` |
| `litersEst` | 官方容积未对照则为 null |
| `extenderOn` | 该时段是否见过增程开启/发电 |

### 1.6c FillEvent 充电 / 加油

按次记录。**不是**官方账单。耗油差分仍在用量里，不进花费列表。

| 字段 | 说明 |
|---|---|
| `kind` | `charge` / `refuel` |
| `source` | `auto` / `manual` |
| `status` | `draft`（待补花费）/ `recorded`（有实付） |
| 电量/油量/里程 | 快照预填，可改 |
| `energyKwh` / `liters` | 用户填的度数或升数，可选 |
| `paidCny` | 实付；有才入账 |
| `unitCny` | 计算：实付/度 或 实付/升 |
| `spend` / `capacity` | 能耗 API 计算。容量用填了度数且 SOC 差≥5% 的充电取中位数 |

充电自动草稿必须 SOC 上升≥2% 且插枪或充电状态。枚举未对照前充电请手补。

### 1.7 SyncJob 同步任务

| 字段 | 说明 |
|---|---|
| `id` | |
| `bindingId` | |
| `kind` | `manual` / `cron` / `admin` / `unknown`（旧行迁移） |
| `status` | `running` / `ok` / `auth_failed` / `upstream` / `decode` |
| `startedAt` / `finishedAt` | |
| `errorPublic` | 可给车主看的短句 |
| `errorInternal` | 仅管理员，仍须脱敏 |

---

## 2. 表（SQLite）

实际表名（不要用文档早期的 `admin_users` / `vehicle_bindings` 那种复数建议名）：

| 表 | 说明 |
|---|---|
| `binding` | 车辆绑定；VIN/refresh/access 密文列，对外只给脱敏与有/无 |
| `owner_session` | 车主会话，只存 token 哈希 |
| `snapshot` | 解码后快照 JSON，按车追加 |
| `energy` | 官方能耗，一车一行覆盖 |
| `sync_job` | 同步历史，主键 `id`，不是 `binding_id` |
| `admin_user` / `admin_session` | 管理员；会话带 username |
| `app_settings` | `cron_sync` / `cors_origins` / `snapshot_keep` / `stale_after_sec` |
| `fill_event` | 充电/加油按次记录 |

凭证不单独成表，加密后放在 `binding` 的 cipher 列。

原则：

- 每张表都有 `created_at` / `updated_at` / `deleted_at`（unix 秒）。gdb Insert/Update/Delete 自动写，查询自动带 `deleted_at=0`。解绑会话是软删。
- `fetched_at` / `started_at` / `finished_at` / `loc_reported_at` 是业务时刻，不是行生命周期，仍按语义赋值
- 快照 / 能耗 payload 与对外 JSON 的时间用北京时间 `YYYY-MM-DD HH:mm:ss`；读入仍接受旧 RFC3339。管理端库表浏览同样返回墙钟字符串，不把 unix 秒丢给浏览器换算
- 原始官方包默认 **不存**
- 不建「轨迹点」表

---

## 3. 对外 JSON 原则

- 车主 API：快照 + 能耗 + 绑定状态；VIN 用 `vinMasked`
- 管理员 API：绑定列表、同步、计数、脱敏表浏览；同样 `vinMasked`；无凭证字段、无完整 VIN
- `decodeWarning` 要给前台，避免把「未换算的原始值」当已换算公里
- 时间字段见 D22：北京时间墙钟，前台不换算
