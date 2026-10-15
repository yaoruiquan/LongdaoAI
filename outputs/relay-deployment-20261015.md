# 中转发布记录：2026.10.15-1 / 2026.10.15-2

时间均为北京时间（Asia/Shanghai）。用户已明确要求先提交本批开发，再部署生产中转并继续真实图片与账务验收。

## 当前发布：2026.10.15-2

- 活跃色 green，镜像 `ghcr.io/yaoruiquan/sub2api:2026.10.15-2`，源码提交 `5e6c4439308f61ad9afafa1cbf152b2f5f20ab04`。旧 blue `2026.10.15-1` 已停止保留。
- 镜像 ID `sha256:0c55d6e63a0cccf0222e199cfcff909c6edfde51d26b255a418f328daf5f830b`，干净源码归档 SHA256 `6583867e5c8dcc9a532c798929daac771d1b0c98c65e703ecf117bc06b32679b`。
- 发布目录 `/opt/longdao/releases/2026.10.15-2/`，构建日志 `build.log`。未推送 Git 或镜像仓库。
- `compose up --no-deps --wait sub2api-green` 健康后，优雅停止旧 blue；未运行 switch.sh、未覆盖 Caddy、未重建 PG/Redis。`.env IMAGE_TAG=2026.10.15-2`、`.active_color=green` 已持久化。
- `/health`、`/`、`/admin/accounts` 均 HTTP 200，新 green healthy、重启 0、无 OOM。未部署生产 SEP、未运行其迁移或开启其全局真实执行。

本次修复报价与实际扣费不一致：报价复用当前 key/group 生效的既有确定性图片按次算法，携带 `pricing_authority=sub2api_customer_billing`；来源仍诚实保留 fallback，不宣称模型官方价格。实际扣费算法未变，未修改任何分组商业价格或倍率。未知/token/default 价格仍不授权图片执行。

验收 `live20261015_8b6cf22a0241`：经隔离本地 SEP 真实生成/参考编辑各一张，1024x1024/medium/单图；每笔原请求账单均 `settled`、USD `0.2934600000`，合计 `0.5869200000`。价格为当前基础算法 `0.134 × 图片倍率2.19`；未填充原为 NULL 的图片基础价格。SEP 测试个人钱包按冻结汇率7.2合计记账 CNY `4.225824`。HTTP 和 Electron 正常退出重启、原回执显式恢复/预览/导出均通过，没有增加模型请求或重复账务。此项不证明生产 SEP 自然购买员工、Pi 真实决策或人工审批。

第二版回滚备份位于 `/opt/longdao/releases/2026.10.15-2/rollback/`（私有 env、active_color、Caddy、数据库及 appdata）。数据库 dump 已通过 `pg_restore -l`，数据库与 appdata 已复制本机 `/Users/yao/.codex/backups/longdao/2026.10.15-2/rollback/`，两端 SHA256 一致：

- `sub2api.dump`：`1c6daa8a38cb932bfe70f935ffaa67658a3448e88aa1ed151a093ec270e14189`。
- `app-data.tar.gz`：`516f5be34cfa967e92abb024b5737cd6391df45f8949f6f52f2bd19b933eae2d`。

回滚启动保留的 blue 第一版、验证健康后停止 green，再恢复 `.env` 与 `.active_color`，验证实际流量。无需删除卷或恢复数据库。以下第一版记录为历史状态，不代表当前仍阻塞或零收费。

## 首次发布结果（历史）

- 2026-10-15 14:40:05 完成蓝绿切换，活跃色为 blue。
- 镜像：`ghcr.io/yaoruiquan/sub2api:2026.10.15-1`。
- 源码提交：`a2d9020cc9c9386934f05546dbb2d67be6a92628`。
- 镜像 ID：`sha256:203ce020a5e87a87f902214f551184959510ba442fd979601783d9498c1f72b1`。
- 从上述已提交源码的干净归档构建，未推送 Git 或镜像仓库；其他会话未提交改动不纳入镜像。
- 源码归档 SHA256：`cccc48cecf146645a5bf4f19baf3c97e769ef4d2e693fc99e6657a83fae04bca`。
- 发布目录：`/opt/longdao/releases/2026.10.15-1/`，构建日志为 `build.log`。
- `.env` 的 `IMAGE_TAG` 已持久化为新版本，`.active_color` 更新为 blue。

## 上线范围

新增 `POST /v1/sub2api/images/pricing` 与 `GET /v1/sub2api/usage?request_id=...`，提供当前 key/group 范围内的明确图片报价和原请求已提交账务证据。不以共享 key 汇总费用变化推算单请求费用。

使用既有 Compose 项目显式启动 blue，带 `--no-deps`；确认健康后切换并优雅停止旧 green。旧容器及镜像保留，未删除数据卷。新旧源码迁移目录完全一致，没有另行运行独立数据库迁移。

未重建共享 PostgreSQL、Redis、Caddy；保留既有线上 Caddy 修复，未热加载或覆盖代理配置。未部署 SEP 生产服务、执行 SEP 数据库迁移或开启 SEP 全局真实执行。

## 验证与阻塞

2026-10-15 14:46:23 复核：

- `/health`、`/`、`/admin/accounts` 均 HTTP 200。
- blue 健康；blue、旧 green、Caddy、PostgreSQL、Redis 重启计数均 0，无 OOM。
- 用户授权测试 key 的 `gpt-image-2`、`1024x1024`、`medium`、单图生成/编辑询价均 HTTP 200，但 `pricing_status=unavailable`、金额 NULL、来源 fallback。
- 诊断 request ID 用量查询 HTTP 200 / pending / 无金额；此查询不对应真实订单，不能当作结算成功。
- 仅以只读事务核查该 key：图片独立倍率 2.19，1K/2K/4K 基础单价均 NULL，未关联渠道。

真实收费请求为 0。已向用户确认生成与编辑的收费标准，未擅自写价格或绕过报价。真实生成、参考编辑、本地保存/恢复、真实单请求结算及 Electron 实机验收仍未完成。

## 备份与回滚

远端私有备份目录：`/opt/longdao/releases/2026.10.15-1/rollback/`。

数据库 custom-format dump 已通过 `pg_restore -l` 校验；应用数据、生产 env、活跃色和 Caddy 配置均已备份。目录权限 700、文件 600。数据库及应用数据另复制到本机：`/Users/yao/.codex/backups/longdao/2026.10.15-1/rollback/`，两端 SHA256 相同：

- `sub2api.dump`：`8c51bcaa445f43b45f0d4015a52a83c9c91676d31bae25961de3f1234fe0945f`。
- `app-data.tar.gz`：`f327a661416d5e4b53b2c9376c7bc8e1e327ea8720928bcf8b3add7989718ba6`。

旧 green 镜像为 `ghcr.io/yaoruiquan/sub2api:2026.10.12-1`，停止但保留。回滚时先启动旧 green、确认健康，再优雅停止新 blue，并恢复 `.active_color` 与 `.env` 的旧 IMAGE_TAG；当前 Caddy 优先 blue，必须停止新 blue 后验证实际流量已回到 green。无需删除卷或恢复数据库。
