# 中转发布记录：2026.10.15-1

时间均为北京时间（Asia/Shanghai）。用户已明确要求先提交本批开发，再部署生产中转并继续真实图片与账务验收。

## 发布结果

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
