# 哪吒官方协议（证据记录）

本文只记录 **已经用本地 HAR 看到的事实** 和 **明确未验证项**。实现官方客户端时只准依赖「已证实」；把「待验证」做成可运行代码必须先补证据并改本页。

更细的抓包条目见本机 `HAR-ANALYSIS.md`（已 gitignore，不进仓库）。HAR 本身含大量个人数据，**不要打开后把内容贴进仓库。**

证据等级：

- **已证实**：请求与响应都在，业务成功
- **静态线索**：官方 JS 里有调用，无成功响应
- **未验证**：需要下一轮实车或新抓包

---

## 已证实的只读能力

根地址（车况/账号）：`https://appapi-pki.chehezhi.cn:18443`  
能耗根地址：`https://api.chehezhi.cn`

业务成功样本：HTTP 200 且 `code = 20000`。

| 接口 | 方法 | 请求 | 响应要点 |
|---|---|---|---|
| `/pivot/mds-api/vehicleAccount/1.0/getCurrentVehicle` | POST JSON `{}` | 当前车、车型、绑定、权限 |
| `/pivot/veh-status/vehicle-status-control/1.0/getAppVehicleData` | POST 表单 `vin, types`（样本 `types` 空） | 13 组车况 |
| `/pivot/veh-status/vehicle-status-control/1.0/getAppVehicleDataByFields` | POST Query `vin, fields` | 按字段取子集 |
| `/pivot/rc-api/appConfig/1.0/getConfig` | POST Query `configKey` | 车控 **能力配置**（不是命令成功） |
| `/pivot/vehicle-data-api/vehicleEnergyConsumption/1.0/findEnergyConsumptionStatistics` | POST 表单 `vin, type` | 能耗汇总 |
| `/pivot/vehicle-data-api/vehicleEnergyConsumption/1.0/queryEnergyConsumptionByVin` | POST 表单 `vin, type` | 按日记录 |

样本车型：哪吒 L，`modelCode = EP32`。

### 换票（探活，2026-09-10）

HAR 里仍没有这条请求。用表单探活成功，**不是**从抓包抄来的。

| 项 | 值 |
|---|---|
| 方法 / URL | `POST https://appapi-pki.chehezhi.cn:18443/customer/account/info/refreshApiToken` |
| Content-Type | `application/x-www-form-urlencoded` |
| 请求体 | `refreshToken=<refresh_token>`（字段名是 camelCase） |
| 未带 | `sign` / `appId` / `Authorization` / 客户端证书 |
| 成功 | HTTP 200，`code=20000`，`success=true`，`data.access_token` / `refresh_token` / `token_type` / `expires_in` |
| 假 token | HTTP 200，`code=41141`，`登录状态已过期，请重新登录` |

随后用新 `access_token` 调 `getCurrentVehicle`（JSON `{}`，只带 `Authorization: Bearer`，无 `sign`）同样 `code=20000`。不证明所有接口都可以不签。

登录相关（**产品不实现短信登录**，仅作协议知识）：

- 发码、验证码登录在 HAR 中业务成功，返回 `access_token` / `refresh_token` / `expires_in`
- 样本 `expires_in ≈ 604799`（约七天），当作变量不要写死
- HAR **没有** refresh 换票的 HTTP 请求。实锤见下方「换票（探活）」

---

## 车况结构（已证实键名）

`getAppVehicleData.data` 下有：

```
vehicleBasic  energyConsumption  seatStatus
doorCoverStatus  airconditionStatus  windowStatus
tyreStatus  vehicleConnection  lockStatus
enduranceStatus  vehicleExtend  chargingStatus
lampStatus
```

### 已对照官方 App「车辆详情」（2026-09-10 只读拉取 + 车主截图）

车头朝上：左前 / 右前 / 左后 / 右后。

| 原始 | 换算 | 对照 |
|---|---|---|
| `tyreStatus.tireLeftFrontPress` 等 | `/55` → bar（解码层截 2 位，App 显示时截 1 位） | 135→2.45、136→2.47、133→2.41、138→2.50 |
| `tyreStatus.tireLeftFrontTemp` 等 | `-50` → ℃ | 73→23、72→22 |
| `airconditionStatus.airInTemp` | `(raw-110)/2` 四舍五入 1 位 → 车内 ℃ | 156→23.0 |
| `airconditionStatus.airOutTemp` | 同上 → 车外 ℃ | 159→24.5 |
| `vehicleBasic.mileage` | `/10` → km | 12345→1234.5，App 显示 1234km |
| `enduranceStatus.powerResidueMileage` + `fuelMileageRemaining` | 各 `/10` 再相加 | 1240+2140→338，App「续航里程 338km」 |

电压、12V 已证实（见下「GB/T 32960 假设边界」）；电流仍是强候选。

### 两份快照与截图的时间轴（避免再混淆数据源）

存在两份相隔约 40 小时的样例上报，**字段联动自洽**，用来说明是同一辆停放车的读数差，而不是抄写错。开源文档只用假里程与 round 时间戳：

| | 快照 A（testdata 报文） | 快照 B（对照截图） |
|---|---|---|
| `reportTime` | 1767225600000 | 1767369180000（晚 39h53m） |
| `soc` | 50 | 49 |
| `totalVoltage` | 3545 | 3543 |
| 纯电续航 | 1270 | 1240 |
| `mileage` | 12345 | 12345（未行驶） |

SOC 降 1%、静置压降 0.2V、续航降 3km、里程不变 —— 四字段自洽。**3545 与 3543 都是有效读数，只是时刻不同**，不要把时刻差误判为抄写错。

**对照截图渲染的是快照 B，不是快照 A。** 证据：截图（胎压 2.4/2.4/2.4/2.5、胎温 23/22/22/23、气温 23.0/24.5）与快照 B（135/136/133/138、73/72/72/73、156/159）三组四角全中；与快照 A（135/135/133/136、71/71/71/72、154/156）处处差 1~2 单位。快照 A 与截图的 ±2℃ **不是公式矛盾，是 40h 环境漂移**。后续对照一律以快照 B ↔ 截图为准，不要拿快照 A 对截图然后怀疑公式。

### GB/T 32960 假设边界（先验，非免验证权威）

GB/T 32960.3 是**车 → 国家监测平台的二进制帧协议**；本接口是**车 → 哪吒云的 HTTPS JSON**（OEM CAN 信号名）。国标只能作缩放先验，不能当法定约束 —— 本协议的温度约定就与国标不同：

- 国标温度偏移 −40，但胎温实测 `73→23℃` 只有 **−50** 成立（四角全中），**国标 −40 被当场证伪**。
- 国标电池总电压分辨率 0.1V（即 ÷10），与本接口 `totalVoltage`/`batVoltage[0]` 冗余互验 + 量纲唯一一致 → **电压 ÷10 升级为已证实**。
- 国标电池总电流分辨率 0.1A、偏移 −1000A（raw=10000 → 0.0A）→ 仅作 `totalCurrent` 的**强候选先验**，仍输出 null。理由：本协议族哨兵本就不统一（65535/65534/255/10000 并存，见 `averageConsumptionPerKm`、`batElectricity[0]` 同为 10000），10000 也可能是「未采集」哨兵；且不得用单样本未验证的 `runMode`/`chargeStatus` 循环互证。**判别实验**：充电/行驶/能量回收三态路试 —— 回收时 raw 应 <10000，一次路试双向验证符号与分辨率。

门/锁/窗（车主确认当时全部关闭，对照同一份只读拉取）：

| 原始 | 关闭样本 | UI |
|---|---|---|
| `doorCoverStatus.*`、`lockStatus.*` | `0` | `closed` |
| 四窗 `windowStatus.*WinStatus` / `rightAfterWinLockStatus` | `16` | `closed` |
| `skyWindowStatus` | `0` | `closed` |

开门/开窗的原始值未见，其它数字保持 `unknown`，不把非 0 当成开。

### 仍为候选 / 待判别实验

| 原始 | 档位 | 说明 |
|---|---|---|
| `vehicleBasic.soc` | ✅ 已证实 | 电量 %，本次 49；与 powerPercentage/电压/续航联动自洽 |
| `enduranceStatus.powerPercentage` | ✅ 已证实 | /100，本次 4900→49 |
| `enduranceStatus.estimateFuelPercent` | ⚠️ 候选 | 油量 %，本次 24 |
| `enduranceStatus.estimateFuelLevel` | ⚠️ 候选 | ≈升，本次 11；与 % 线性自洽但油箱标称未定，等加油跳枪事件判别 |
| `vehicleBasic.totalVoltage` | ✅ 已证实 | /10，本次 3543→354.3V（量纲唯一+冗余互验+国标惯例） |
| `vehicleExtend.battery12Voltage` | ✅ 已证实 | /10，样本 127→12.7V（量纲唯一） |
| `vehicleBasic.totalCurrent` | ⚠️ 强候选 | GB/T 32960 先验 (raw−10000)/10，**仍输出 null**；等充/放/回收三态路试 |
| `energyConsumption.averageConsumptionPerKm` | ⚠️/🚫 | 本次 10000，可能是第四种哨兵，单样本 |
| 车窗非 16 / 天窗非 0 / 门盖非 0 | 🚫→⚠️ | 开侧枚举未见，保持 unknown |
| `chargeStatus` vs `powerChargeStatus` | 🚫 | 不同枚举，不可共用映射 |
| `vehicleConnection.reportTime` | ✅ 已证实 | 毫秒时间戳，用作 `reportedAt` |
| `todayEnergyConsumption` 等为 65535 | ✅ 哨兵 | 当未知 |

未发现可公开宣称的 SOH、电芯压差列表、充电记录列表。

`vehicleExtend.lng/lat` **存在且已采集**（决策 D17 修订）。解码器 ÷1e6 得 WGS-84 经纬度，仅保留最新一个点，车主 App 与管理员 Web 均可见；不存历史轨迹。开源示例一律用明显假值，如 `lng=116000000, lat=40000000` → 116.0°E / 40.0°N；**禁止**把车主真实坐标写入文档、测试或 testdata。

---

## 能耗（已证实）

字段：`vin, countTime, totalConsumesEnergy, drivingConsumesEnergy, airConditionersConsumesEnergy, recoveryConsumesEnergy, windResistanceConsumesEnergy, attachmentConsumesEnergy`。

- HAR 里响应可能是 Base64 + Brotli，客户端要按真实 Content-Encoding 处理，不能只 `JSON.parse` HAR 文本
- 官方 H5 用 `type = 选项索引 + 1` → 1 周 2 月 3 年；只实测过 1
- UI 单位线索：kWh；**无油升、无纯电里程**
- 样本 CORS 允许 `https://h5-battery.chehezhi.cn`，与自建前端无关（我们走服务端）
- 样本中 Authorization 为空仍成功 **只描述那一次**，自建服务仍必须做归属校验，不得查询任意 VIN

---

## 未验证（阻塞或推迟）

| 项 | 说明 |
|---|---|
| **sign** | 64 hex，算法未知。换票与一次 `getCurrentVehicle` 探活未带 sign 仍成功，禁止据此编造签名器 |
| 客户端证书 | `applyCert` 样本 400 `No required SSL certificate was sent`；不代表所有车况接口都要 mTLS，也不代表能从 HAR 推出私钥 |
| 多车列表 | 只有当前车 |
| `getVehicleHealth` | 仅 JS 线索 |
| 车控下发 | 配置有空调/门窗等标识，**零条**命令+回执 |
| 官方短信登录复刻 | 产品禁止做 |

换票一旦实锤，把 host、method、path、content-type、请求字段、响应字段（脱敏）追加到本节「已证实」，并去掉「未验证」对应行。

---

## 实现时禁止

- 把第三方站点 path 或 `{userId}` 模板当哪吒官方
- GET 默认拉车况（已证实是 POST 表单）
- 用浏览器 CORS 代理直连
- 把数字钥匙 JSON（另一信封 `code=200, businessObj`）当主接口解析器
- 在开源测试数据里放真实 VIN / token / 坐标
