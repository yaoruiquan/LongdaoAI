# Sub2API 文档中心

这里是 Sub2API / 龙道 AI 中转站的项目文档目录，用来沉淀上线部署、产品改造、支付接入、图片任务、合规说明和运维知识。

如果只是想快速找到该看的文档，先看下面的「快速入口」；如果要维护或补充文档，参考最后的「维护约定」。

## 快速入口

| 你要做什么 | 推荐阅读 |
| --- | --- |
| 从零了解怎么搭一个 AI API 中转站 | [搭建 AI 中转站指南](./搭建AI中转站指南.md) |
| 准备把龙道 AI 部署到服务器 | [服务器上线前准备审计](./SERVER_DEPLOYMENT_READINESS.md)、[服务器部署实录](./龙道AI服务器部署实录.md) |
| 按正式上线标准拆需求和门禁 | [服务器上线准备 Spec](./specs/2026-07-15-longdao-server-launch-spec.md) |
| 看产品改造路线、阶段和验收口径 | [Token 中转站改造路线图](./TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md) |
| 配置内置支付系统 | [支付系统配置指南（中文）](./PAYMENT_CN.md)、[Payment System Configuration Guide](./PAYMENT.md) |
| 对接外部支付系统或 Sub2ApiPay | [Admin 支付集成 API](./ADMIN_PAYMENT_INTEGRATION_API.md) |
| 规划支付宝 / 微信 / 易支付接入 | [龙道 AI 支付接入指南](./payment-integration-guide.md) |
| 使用异步图片生成 / 编辑接口 | [异步图片任务](./ASYNC_IMAGE_TASKS.md) |
| 使用 Gemini / Vertex 批量图片生成 | [Batch Image MVP](./BATCH_IMAGE_MVP.md) |
| 给运营或管理层做汇报 | [龙道 AI 领导汇报文档](./龙道AI-领导汇报文档.md) |
| 查部署、Docker、SSH、GitHub、DNS 等基础知识 | [部署知识库](./knowledge-base/README.md) |

## 文档分类

### 1. 上线与部署

| 文档 | 内容 | 适合场景 |
| --- | --- | --- |
| [SERVER_DEPLOYMENT_READINESS.md](./SERVER_DEPLOYMENT_READINESS.md) | 上线前审计、P0/P1/P2 清单、封闭内测与公开收费边界 | 服务器上线前做风险检查 |
| [specs/2026-07-15-longdao-server-launch-spec.md](./specs/2026-07-15-longdao-server-launch-spec.md) | 服务器上线准备 Spec，覆盖发布门禁、迁移、备份、回滚、支付和风控要求 | 按工程标准推进正式上线 |
| [龙道AI服务器部署实录.md](./龙道AI服务器部署实录.md) | 从 SSH 登录到 Docker、域名、环境变量、HTTPS 验证的完整实操记录 | 复盘或复用已有部署步骤 |
| [搭建AI中转站指南.md](./搭建AI中转站指南.md) | VPS、域名、上游 API、费用和搭建流程概览 | 给新成员或非工程角色快速解释整体方案 |

相关部署脚本和生产运维说明在仓库的 [deploy](../deploy) 目录，尤其是 [deploy/production/README.md](../deploy/production/README.md) 和 [deploy/production/OPS_README.md](../deploy/production/OPS_README.md)。

### 2. 产品路线与业务材料

| 文档 | 内容 | 适合场景 |
| --- | --- | --- |
| [TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md](./TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md) | Token/API 中转平台改造路线、阶段目标、验收标准和下一步行动 | 规划产品和工程迭代 |
| [龙道AI-领导汇报文档.md](./龙道AI-领导汇报文档.md) | 产品定位、能力覆盖、技术瓶颈、成本投入和资源诉求 | 汇报、立项、资源沟通 |
| [龙道AI中转站成本核算.xlsx](./龙道AI中转站成本核算.xlsx) | 成本测算表格 | 估算服务器、上游和运营成本 |
| [plans/2026-07-14-longdao-ai-phase-1-frontend.md](./plans/2026-07-14-longdao-ai-phase-1-frontend.md) | 第一阶段前端改版执行计划 | 回看前端阶段实施范围 |
| [plans/2026-07-14-longdao-ai-phase-1-frontend-design.md](./plans/2026-07-14-longdao-ai-phase-1-frontend-design.md) | 第一阶段前端设计计划 | 回看视觉和交互设计约束 |

### 3. 支付与充值

| 文档 | 内容 | 适合场景 |
| --- | --- | --- |
| [PAYMENT_CN.md](./PAYMENT_CN.md) | 内置支付系统中文配置指南，覆盖 EasyPay、支付宝、微信、Stripe、Webhook 和迁移 | 后台配置自助充值 |
| [PAYMENT.md](./PAYMENT.md) | 内置支付系统英文配置指南 | 英文环境或对外文档 |
| [ADMIN_PAYMENT_INTEGRATION_API.md](./ADMIN_PAYMENT_INTEGRATION_API.md) | 外部支付系统调用 Sub2API Admin API 的中英双语说明 | 支付成功后通过兑换码或余额接口入账 |
| [payment-integration-guide.md](./payment-integration-guide.md) | 支付宝官方直连与第三方聚合平台路线对比、申请步骤和风险提示 | 选择支付接入路线 |

支付相关文档分两类：`PAYMENT_CN.md` / `PAYMENT.md` 面向 Sub2API 内置支付；`ADMIN_PAYMENT_INTEGRATION_API.md` 面向外部支付系统回调入账。

### 4. 图片能力

| 文档 | 内容 | 适合场景 |
| --- | --- | --- |
| [ASYNC_IMAGE_TASKS.md](./ASYNC_IMAGE_TASKS.md) | OpenAI 兼容图片生成 / 编辑异步任务接口、对象存储配置、轮询和清理机制 | 避免长连接超时，接入异步图片任务 |
| [BATCH_IMAGE_MVP.md](./BATCH_IMAGE_MVP.md) | Gemini / Vertex 批量图片生成接口、任务状态、输出下载和限制 | 大批量图片生成或批处理工作流 |

### 5. 合规与声明

| 文档 | 内容 | 适合场景 |
| --- | --- | --- |
| [legal/admin-compliance.zh.md](./legal/admin-compliance.zh.md) | 中文版部署与运营合规承诺 | 管理员确认责任边界 |
| [legal/admin-compliance.en.md](./legal/admin-compliance.en.md) | English deployment and operation compliance commitment | English-facing compliance notice |

### 6. 部署知识库

知识库是给部署和运维过程补课用的材料，入口见 [knowledge-base/README.md](./knowledge-base/README.md)。

| 文档 | 内容 |
| --- | --- |
| [knowledge-base/vps-and-network.md](./knowledge-base/vps-and-network.md) | VPS 选型、域名 DNS、HTTPS、Caddy 反向代理和防火墙基础 |
| [knowledge-base/docker-compose-basics.md](./knowledge-base/docker-compose-basics.md) | Docker Compose、镜像、容器、网络、数据卷、健康检查和常用命令 |
| [knowledge-base/deployment-flow.md](./knowledge-base/deployment-flow.md) | 备份、迁移、部署、验证、回滚和版本升级流程 |
| [knowledge-base/ssh-key-auth.md](./knowledge-base/ssh-key-auth.md) | SSH 密钥认证、权限、sshd_config、代理干扰和常见问题 |
| [knowledge-base/github-auth.md](./knowledge-base/github-auth.md) | GitHub PAT、OAuth Token、workflow scope 和 credential helper |
| [knowledge-base/git-branch-management.md](./knowledge-base/git-branch-management.md) | Fork、tag、worktree、rebase、重新移植和分支策略 |
| [knowledge-base/api-relay-architecture.md](./knowledge-base/api-relay-architecture.md) | API 中转链路、上下游、模型映射、Codex 端点和计费设计 |

## 推荐阅读路径

### 准备封闭内测上线

1. [SERVER_DEPLOYMENT_READINESS.md](./SERVER_DEPLOYMENT_READINESS.md)
2. [specs/2026-07-15-longdao-server-launch-spec.md](./specs/2026-07-15-longdao-server-launch-spec.md)
3. [龙道AI服务器部署实录.md](./龙道AI服务器部署实录.md)
4. [deploy/production/README.md](../deploy/production/README.md)
5. [deploy/production/OPS_README.md](../deploy/production/OPS_README.md)

### 准备公开收费上线

1. [TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md](./TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md)
2. [PAYMENT_CN.md](./PAYMENT_CN.md)
3. [payment-integration-guide.md](./payment-integration-guide.md)
4. [ADMIN_PAYMENT_INTEGRATION_API.md](./ADMIN_PAYMENT_INTEGRATION_API.md)
5. [legal/admin-compliance.zh.md](./legal/admin-compliance.zh.md)

### 新成员了解项目背景

1. [搭建AI中转站指南.md](./搭建AI中转站指南.md)
2. [knowledge-base/api-relay-architecture.md](./knowledge-base/api-relay-architecture.md)
3. [TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md](./TOKEN_RELAY_CUSTOMIZATION_ROADMAP.md)
4. [龙道AI-领导汇报文档.md](./龙道AI-领导汇报文档.md)

## 维护约定

- 新增文档优先放在最贴近主题的目录：上线需求放 `specs/`，计划放 `plans/`，基础知识放 `knowledge-base/`，合规文本放 `legal/`。
- 新增或重命名文档后，同步更新本 README 和相关子目录 README。
- 面向执行的文档尽量写清楚前提、步骤、验证方式和回滚方式。
- 涉及支付、资金、账号、密钥和生产环境的文档要明确风险、权限边界和不可提交的敏感信息。
- 历史实录类文档保留日期和背景，不直接覆盖成最新状态；最新决策应沉淀到路线图、Spec 或 README。
