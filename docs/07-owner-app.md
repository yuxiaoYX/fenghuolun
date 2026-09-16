# 车主前台（uni-app x 蒸汽模式）

路径：`apps/owner`。这是车主 **唯一** 产品界面。

## 技术锁定

- 工程类型：**uni-app x**（uvue 页面；蒸汽模式逻辑层用 js/ts，uts 主要用于原生插件）
- 编译：**蒸汽模式** = Vue Vapor + uni-app x 原生渲染管线
  - **不是** Steam 游戏视觉
  - **不是** 蒸汽朋克皮肤
  - **不是** 普通 Vite 项目里的 Vue Vapor 实验开关
- 官方开关（已写入 `apps/owner/manifest.json`）：`uni-app-x.vapor = true`，并设 `styleIsolationVersion: "2"`、`vapor-render-target: "bytecode"`。HBuilderX 可视化首页也可勾选「蒸汽模式」。说明：[蒸汽模式](https://doc.dcloud.net.cn/uni-app-x/app-vapor.html)
- 仅组合式 API；文字必须包在 `<text>` 内；需要 **HBuilderX 5.21+**
- 目标端优先级：App-Android → App-iOS → H5。H5 目前按官方说明以 VDOM 运行
- 禁止：把已删除的根目录 `src/*.vue` 复制进来；禁止在前台实现官方签名与换票

## 信息架构

底栏五项，对齐 `docs/wireframes/hifi-v2.html`。车控是灰色占位，**禁止**发写命令（D12）。

| Tab | 页面 | 内容 |
|---|---|---|
| 车况 | `pages/now` | 一屏总览：俯视车图、胎压胎温、门/后备箱、纯电/燃油续航、温度、充电状态；解锁滑轨与锁车按钮禁用；更多读数（12V、车窗、位置、同步于/上报于）在下方折叠 |
| 车控 | `pages/ctrl` | 温控 / 门窗充电 / 滑动危险操作的布局占位，全部「尚未开通」，点击不发请求 |
| 能耗 | `pages/energy` | 电（官方 kWh）和油（油量差分）并列；充电/加油按次实付；右下角 ＋ 默认补充电 |
| 电池 | `pages/battery` | 12V、同电量包电压、电流；SOH 与满电容量估算无证据则空态 |
| 我的 | `pages/more` | 车辆资料与备注、绑定/更换 `refresh_token`、解绑、充能花费入口、车况历史、关于与免责；导出仍占位 |

绑定页 `pages/bind`、历史页 `pages/history`、充能页 `pages/fill` 不进底栏。远程启动在车况页底部占位「暂未开放」。

## 绑定页

- 一个多行输入，标签 **refresh_token**
- 说明：从官方哪吒 App 自行取得；本服务用它换票，不会在界面再要网关和签名
- 成功：进入「车况」
- 失败：`token_invalid` → 「请重新填写 refresh_token」；会话不建立
- 绑定只接受官方 `refresh_token`；后端必须设置 `FENGHUOLUN_TOKEN_KEK`
- 本地只存本服务会话，不存官方 token

## 展示纪律

1. `null` / `unknown` → 「—」，不编数字
2. 车况标题行用相对时间（由 `reportedAt`，否则 `fetchedAt`）。「同步于 / 车辆上报于」完整墙钟放在「更多读数」。这两个字段后端已是北京时间，前台直接展示，禁止再换时区。能耗没有 `reportedAt`，只写同步于，不要补「车辆上报于 —」
3. 后端按 `stale_after_sec`（默认 7200）计算 `stale`（不落库）；车况相对时间与电池时效条变色。无 `reportedAt` 不标陈旧
4. `online` 已知则写在完整时效行旁（在线/离线）；未知不要写成离线
5. `decodeWarning` 用低调提示，例如「部分单位尚未对照实车，已隐藏」
6. 增程块：`extender == null` 则燃油续航列不渲染
7. 不用地图组件；车图是只读俯视示意，不接受车控点击
8. 充电状态未知时写「—」和「插枪状态未知」，**禁止**显示成「未充电 / 未插枪」
9. 挡位无字段，写「—」，不假装 P 挡

## 与后端

- 基础 URL：`apps/owner/common/config.uts` 的 `API_BASE`（开发默认 `http://127.0.0.1:8088`；Android 模拟器改 `http://10.0.2.2:8088`）
- 只调用 `docs/06-backend.md` 的 owner API
- UTS 网络层统一处理 `ok/error.code`
- `token_invalid` → 拉回绑定页

## 视觉

高保真 v2（设计稿 `docs/wireframes/hifi-v2.html`）。运行时 token 在 `apps/owner/App.uvue` 的 CSS 变量；`uni.scss` 只作对照：

- 浅色背景 `#EEF2F6`，面板 `#FFFFFF`；深色背景 `#070B10` / `#0B1017`，面板 `#131C27`
- 浅色文字 `#15202C` / `#3D5064`；深色文字 `#EDF4FA` / `#B7C9D8`
- 品牌火 `#FF6A3D`，电 `#2FE08D`，油 `#F5A524`，警示随主题略有不同

默认浅色「晨雾提车」，只在「我的 → 主题」切换深色「座舱夜航」。选择存在本机 `fhl_theme`。导航栏首屏颜色在 `pages.json` 写死浅色，避免 `@theme` 变量未解析时闪黑；窗体底色走 `theme.json` 的 `@bgColor` / `@bgColorContent`，运行时再 `setAppTheme`、`setNavigationBarColor`、`setTabBarStyle`，页面根节点按窗口高度铺当前主题色。导航栏用系统原生栏（不是 custom）。

uni-app x 的 ucss 不能用 `linear-gradient` / `grid` / `::before`，实现时用纯色、flex、真实节点代替。不要做成 Steam 商店皮肤。

## 蒸汽模式约束（实现时核对）

蒸汽 / Vapor 编译通常对动态模板、部分 Vue 写法有限制。约定：

- 蒸汽模式页面用组合式 + js/ts；uts 留给原生插件。少用过于动态的 `any`
- 列表用官方推荐写法
- 不引入依赖浏览器 DOM 的 npm 组件库
- 若某端尚不支持蒸汽模式：该端降级配置必须写进 `apps/owner/README.md`，不得默默全项目关闭蒸汽模式

## 页面完成标准（第 3 期打磨）

- 五 Tab + 绑定页按 `docs/wireframes/hifi-v2.html` 的信息架构
- 蒸汽模式已声明；没有官方 host、没有写死 token
- 未知值「—」；锁/充电/门窗/挡位不编状态；增程块 `extender == null` 不渲染燃油列
- 「车况 / 能耗 / 电池 / 我的」都有同步于或上次同步（车况完整时间在更多读数）
- 远程与车控全部灰色占位，点击只 toast「尚未开通」，不发车控
- 主按钮 / 图标按钮 / 列表项 `min-height` ≥ 44px
- 真机蒸汽模式须用 HBuilderX 打开 `apps/owner` 运行到 App；H5 是 VDOM
