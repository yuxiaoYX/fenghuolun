# 文档索引

本目录是 **风火轮（哪吒 L 车主开源工具箱）** 的权威说明。后续开发、重构、开新对话，都必须先读这里，而不是仓库里残留的原型代码或过时注释。

**阅读顺序（新对话必读）：**

1. [HANDOFF.md](HANDOFF.md) — 下一轮从哪开始
2. [02-decisions.md](02-decisions.md) — 已锁定决策，不得静默改口
3. [01-product.md](01-product.md) — 做什么、不做什么
4. [03-architecture.md](03-architecture.md) — 三端怎么拆
5. [04-roadmap.md](04-roadmap.md) — 分期与完成标准

**实现时再读：**

| 文档 | 用途 |
|---|---|
| [05-data-model.md](05-data-model.md) | 领域对象与库表 |
| [06-backend.md](06-backend.md) | Go 后端、本服务 HTTP |
| [07-owner-app.md](07-owner-app.md) | uni-app x 蒸汽模式前台 |
| [08-admin.md](08-admin.md) | 管理员 Web 后台 |
| [09-protocol-neta.md](09-protocol-neta.md) | 哪吒官方接口：证据 vs 猜测 |
| [10-security.md](10-security.md) | 凭证、HAR、开源边界 |
| [11-conventions.md](11-conventions.md) | 编码与文档约定 |
| [DEPLOYMENT.md](DEPLOYMENT.md) | 1Panel / GHCR / Release 生产部署 |
| `HAR-ANALYSIS.md`（本机，已 gitignore） | 本地 `1.har` 的抓包证据附录，**不进仓库** |

**已作废（不要再当需求来源）：**

- [obsolete/API.md](obsolete/API.md) — 旧 Vite 原型合同
- [obsolete/design.md](obsolete/design.md) — 旧一期配色草稿

产品决策以 `02-decisions.md` 为准。协议事实以 `09-protocol-neta.md` 为准；更细抓包条目见本机 `HAR-ANALYSIS.md`（不进仓库）。二者冲突时：**决策文件赢产品范围，HAR 赢官方字段与路径。**
