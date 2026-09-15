# apps/owner — uni-app x 蒸汽模式

本目录才是 uni-app x 工程根。不要把仓库根 `fenghuolun/` 当 uni-app 项目运行。

官方规则已放入 `.cursor/rules/`（来源：[uni-app-x-ai-rules](https://gitcode.com/dcloud/uni-app-x-ai-rules)）。写 uvue/uts/ucss 时遵守那些文件。

## 本工程约束

- 框架：uni-app x，蒸汽模式（`manifest.json` → `uni-app-x.vapor: true`）。不是 Steam 皮肤。
- 页面：`pages/*.uvue`，组合式 API；文字必须在 `<text>` / `<button>` 里。
- 蒸汽模式 script 可不写 `lang`（js/ts）。UTS 留给原生插件，不要在页面里直接调系统原生 API。
- 仅 flex / 绝对定位；可滚动内容用 `scroll-view`（App 上尤其必要）。
- 不使用 pinia、vuex、i18n、DOM（`window`/`document`，除非 WEB 条件编译）。
- 对象类型用 `type` 不是 `interface` 接对象字面量；不用 `undefined`，用 `null`；条件必须是 boolean。
- 只请求本仓库 Go 后端 owner API。禁止直连官方云、禁止在前台做签名/换票、禁止保存官方 Access Token。
- `appid` 留空，由 HBuilderX 分配。根上必须有 `index.html`（Web/Vue3 编译入口）。

## AI Rules / MCP / uni-agent（能力边界）

- **AI Rules**：`.cursor/rules/*.mdc` 是给 Cursor / 本目录 Agent 读的静态规范，不是在线服务。
- **MCP `@dcloudio/uni-app-x-mcp`**：只负责列出当前工程 easycom 组件。配置在 `.cursor/mcp.json`、`.mcp.json`。要在 Cursor 里手动打开 MCP；DeepSeek Harness 本会话没有接入该 MCP，不要假装调过它。
- **HBuilderX uni-agent（有人口中的 AI Pulse）**：HBuilderX 5+ 右上角 AI。官方已建议用它替代这份历史 rules 仓库。本聊天不是 uni-agent，调不到 HBuilderX 编译器日志。
