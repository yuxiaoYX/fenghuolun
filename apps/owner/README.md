# apps/owner

车主前台。**uni-app x 蒸汽模式**（Vue Vapor 编译，不是 Steam 皮肤）。

## 蒸汽模式如何打开

官方方式：`manifest.json` → `uni-app-x.vapor = true`（HBuilderX 可视化界面首页也可勾选「蒸汽模式」）。

本仓库已写入：

```json
"uni-app-x": {
  "vapor": true,
  "styleIsolationVersion": "2",
  "vapor-render-target": "bytecode"
}
```

依据：[蒸汽模式](https://doc.dcloud.net.cn/uni-app-x/app-vapor.html)、[manifest.uni-app-x.vapor](https://doc.dcloud.net.cn/uni-app-x/collocation/manifest.html)。

约束（实现时遵守）：

- 仅组合式 API，不用选项式、不用 mixin
- 页面用 uvue；蒸汽模式 script 可不写 `lang`（js/ts）
- 文字放在 `<text>` 里
- 运行与打包需要 **HBuilderX 5.21+**（Android 蒸汽模式起点）
- Web / 小程序当前会以 VDOM 跑，官方后续会升蒸汽模式
- 不要在本工程直连官方云
- 本服务地址：`common/config.uts` → `API_BASE`（默认 `http://127.0.0.1:8088`）

`appid` 留空，用 HBuilderX 打开后由 DCloud 分配，不要手写假 appid。

HBuilderX 把本工程当 Vue3 / uni-app x 编译时，**本目录根上必须有** `index.html`（官方 hello-uni-app-x 同样带这个文件；不是管理员后台的 Vite 入口）。缺了会直接报：`请确认您的项目模板是否支持vue3：根目录缺少 index.html`。

## 打开

H5 可以不经过 HBuilderX，在仓库根：

```bash
pnpm owner:h5
```

或 `pwsh -File apps/owner/dev-h5.ps1`。默认 `http://127.0.0.1:5174`（避开 HBuilderX 常用的 5173）。脚本用的是本机 HBuilderX 插件目录里的 `uni` CLI，不是另装一套。

- 终端能看到：UTS / Vite 编译错误、启动地址
- 终端看不到：浏览器里的 `console.log`（仍在开发者工具）
- 真机 App / 蒸汽模式仍须 HBuilderX。本仓库不能代替「运行到手机」；H5 按官方以 VDOM 跑。

HBuilderX 路径：

1. 用 HBuilderX 5.21+ **只打开本目录** `apps/owner`（不要把仓库根 `fenghuolun/` 当成 uni-app 工程运行）
2. 确认项目图标为圆形 U，且蒸汽模式已勾选（`manifest.json` 里 `uni-app-x.vapor: true`，并有 `vueVersion: "3"`）
3. 运行到 Android / iOS / 浏览器
4. 第一次打开后让 HBuilderX 分配 `appid`，不要手写假 appid

## 官方 AI Rules / MCP / uni-agent

官方仓库：[uni-app-x-ai-rules](https://gitcode.com/dcloud/uni-app-x-ai-rules)。说明：[AI Rules 和 MCP](https://doc.dcloud.net.cn/uni-app-x/tutorial/rules_mcp.html)。该仓库 README 已写 **推荐改用 [uni-agent](https://doc.dcloud.net.cn/uni-app-x/ai/)**。

本目录已放入：

- `AGENTS.md`：本工程约束（Codex / 通用 Agent）
- `.cursor/rules/`：uvue / uts / ucss / API 规则（Cursor）
- `.cursor/mcp.json` 与 `.mcp.json`：`npx @dcloudio/uni-app-x-mcp`（只列出 easycom 组件）

在 Cursor 里要到 Settings → MCP 手动打开 `uni-app-x`。DeepSeek Harness 这路对话 **没有** 接入该 MCP，也 **不是** HBuilderX uni-agent。要编译器日志、真机截图、自动修到通过，用 HBuilderX 右上角 AI。
