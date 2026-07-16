# 龙道 AI 生产部署（Docker Compose + 容器化 Caddy）

本目录是龙道 AI 的**生产环境**部署清单，相较上游 `deploy/docker-compose.yml` 做了安全与稳定性收敛：

- 应用镜像固定为自有版本化 tag（禁止 `latest`）。
- 应用**不再直接暴露公网**，对外只经容器化 Caddy 反向代理（HTTPS）。
- PostgreSQL / Redis 均不映射宿主机端口，只在内部网络可达。
- 每个服务都加了 CPU/内存限制与日志轮转，防止资源与磁盘被打爆。
- Caddy 自动签发 Let's Encrypt 证书，HTTP 自动跳转 HTTPS。

## 目录内容

| 文件 | 说明 |
|------|------|
| `docker-compose.yml` | 生产 compose：`sub2api` / `postgres` / `redis` / `caddy` 四服务 |
| `.env.example` | 环境变量模板（只含占位符，复制为 `.env` 后填真实值） |
| `Caddyfile` | 容器化 Caddy 反向代理配置（HTTPS + 流式友好） |
| `README.md` | 本文档 |

> `migrate.sh` / `backup.sh` / `restore.sh` / `deploy.sh` 等运维脚本由发布流程提供，
> 与本清单约定的接口保持一致（见文末“运维脚本接口约定”）。

## 前置条件

1. 一台可从公网访问的服务器，已安装 Docker（含 Compose v2）。
2. 一个**已解析到本机公网 IP 的域名**（Let's Encrypt 需要通过 80/443 完成域名验证）。
3. 防火墙放行入站 **80 与 443**;这是唯一需要对公网开放的端口。
4. 已用 `deploy/build_production_image.sh` 构建好版本化镜像（例如 `longdao/sub2api:v2026.07.15-1`），
   并且部署机能拉到/已有该镜像。

## 快速开始

```bash
cd deploy/production

# 1) 生成环境文件并填写
cp .env.example .env
#   必填项：IMAGE_TAG、LONGDAO_DOMAIN、POSTGRES_PASSWORD、REDIS_PASSWORD、
#          ADMIN_EMAIL、ADMIN_PASSWORD、JWT_SECRET、TOTP_ENCRYPTION_KEY
#   随机密钥用：openssl rand -hex 32

# 2) 校验配置（缺任何必填项都会在此报错）
docker compose config -q

# 3) 启动
docker compose up -d

# 4) 观察日志
docker compose logs -f sub2api caddy

# 5) 访问
#    https://<你的 LONGDAO_DOMAIN>
```

首次启动时 `AUTO_SETUP=true` 会自动建表并创建管理员账号（用 `.env` 里的 `ADMIN_EMAIL` /
`ADMIN_PASSWORD`）。登录后请尽快修改初始密码。

## 如何填写 .env

所有密钥均在 compose 中用 `${VAR:?...}` 强制要求，缺失会导致 `docker compose config` 直接失败。

| 变量 | 必填 | 说明 |
|------|:---:|------|
| `IMAGE_REPO` | 否 | 镜像仓库前缀，默认 `longdao/sub2api` |
| `IMAGE_TAG` | 是 | 版本化不可变 tag，禁止 `latest`，如 `v2026.07.15-1` |
| `LONGDAO_DOMAIN` | 是 | 对外域名，Caddy 据此签发证书 |
| `LONGDAO_ACME_EMAIL` | 否 | Let's Encrypt 注册邮箱（证书到期通知） |
| `POSTGRES_PASSWORD` | 是 | 数据库密码 |
| `REDIS_PASSWORD` | 是 | **生产强制**;启用 Redis 认证 |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | 是 | 首次自动创建的管理员 |
| `JWT_SECRET` | 是 | 固定值，否则重启后登录态失效 |
| `TOTP_ENCRYPTION_KEY` | 是 | 固定值，否则重启后 2FA 失效 |
| `TZ` | 否 | 时区，默认 `Asia/Shanghai` |

其余可选项（Gemini OAuth、URL 白名单、更新代理等）见 `.env.example` 注释。

## 起停与常用命令

```bash
# 启动 / 后台运行
docker compose up -d

# 停止（保留数据卷）
docker compose down

# 停止并删除数据卷（危险：清空数据库/缓存/证书，谨慎）
# docker compose down -v

# 查看状态与健康
docker compose ps

# 拉取新版本镜像并滚动更新（改 .env 的 IMAGE_TAG 后）
docker compose pull sub2api && docker compose up -d sub2api

# 单独重启反代
docker compose restart caddy
```

## 数据库迁移

后端提供独立迁移命令 `sub2api -migrate`，与应用启动解耦：**先迁移成功，再放量新版本**。

```bash
# 用一次性容器执行迁移（--rm 不残留容器）
docker compose run --rm sub2api -migrate
# 等价写法（entrypoint 会在参数为 flag 时自动补上 /app/sub2api）：
# docker compose run --rm sub2api /app/sub2api -migrate
```

迁移返回非零退出码即失败,此时不要启动新版本应用,排查后重跑。发布脚本 `migrate.sh`
已封装该流程。

## 网络边界说明（重要）

```
       公网
        │  仅 80 / 443
        ▼
   ┌─────────┐   longdao-network (bridge, 内部)
   │  caddy  │───────────────────────────────┐
   └─────────┘                                │
        │ reverse_proxy sub2api:8080          │
        ▼                                     │
   ┌─────────┐      ┌──────────┐      ┌───────────┐
   │ sub2api │──────│ postgres │      │   redis   │
   └─────────┘      └──────────┘      └───────────┘
   （无对外端口）    （无对外端口）      （无对外端口）
```

- **唯一对公网开放的是 caddy 的 80/443。** `sub2api`、`postgres`、`redis` 都不映射宿主机端口。
- 应用只能经 Caddy 访问,内网服务间通过服务名（`postgres` / `redis` / `sub2api`）互连。
- 需要临时调试内网服务时,可在对应服务临时加 `ports: ["127.0.0.1:<port>:<port>"]`
  （仅绑定回环,调试完移除),相关注释已写在 compose 文件里。

## 可信代理配置（务必阅读）

应用位于 Caddy 之后,真实客户端 IP 通过 `X-Forwarded-For` / `X-Real-IP` /
`CF-Connecting-IP` 透传。**如果后端不把 Caddy 容器网段列入可信代理,恶意客户端可以伪造
这些头绕过限流与 IP 风控。**

因此上线前请在应用配置(`config.yaml` 或对应环境变量)中,将 **可信代理范围设为
`longdao-network` 的容器网段**(bridge 网络默认 `172.x.0.0/16` 一类的私有段,可用
`docker network inspect longdao-network` 查看实际子网)。仅信任该网段,才应采信转发头里的
客户端 IP。本清单不修改后端代码,此项需在应用侧确认。

若前置了 Cloudflare,则真实 IP 以 `CF-Connecting-IP` 为准,同时应仅信任 Cloudflare 回源 IP 段。

## 流式响应(SSE)说明

Caddy 反代已设置 `flush_interval -1`,逐块立即刷新、不缓冲,保证 SSE / 逐 token 流式
响应实时返回;`transport http` 的读写超时给到 600s 以承载长连接。这是 DEP-002 的验收项。

## 验证清单

- [ ] `docker compose config -q` 无输出且退出码 0(必填项齐全、语法正确)。
- [ ] `docker compose ps` 中四个服务均 `healthy`。
- [ ] `https://<域名>/health` 返回 200,且证书有效(浏览器无告警)。
- [ ] `http://<域名>` 自动 301 跳到 https。
- [ ] 调用一个流式接口,能看到 token 逐步返回而非一次性到达。
- [ ] 后端已配置可信代理网段(见上一节)。

## 运维脚本接口约定(供发布/备份脚本对齐)

| 约定项 | 值 |
|--------|-----|
| Compose 文件路径 | `deploy/production/docker-compose.yml` |
| 环境文件路径 | `deploy/production/.env`(真实值,不进仓库) |
| 应用服务名 | `sub2api` |
| 数据库服务名 / 卷 | `postgres` / `longdao_postgres_data` |
| 缓存服务名 / 卷 | `redis` / `longdao_redis_data` |
| 反代服务名 / 卷 | `caddy` / `longdao_caddy_data`、`longdao_caddy_config`、`longdao_caddy_logs` |
| 应用数据卷 | `longdao_sub2api_data` |
| 内部网络名 | `longdao-network` |
| 镜像变量 | `IMAGE_REPO`(默认 `longdao/sub2api`)+ `IMAGE_TAG`(必填,禁止 latest) |
| 迁移调用 | `docker compose run --rm sub2api -migrate`(先迁移成功再放量) |
| 对外端口 | 仅 caddy 的 `80` / `443` |

标准发布顺序建议:`docker compose pull sub2api` → `migrate.sh`(迁移成功)→
`docker compose up -d`。回滚:把 `.env` 的 `IMAGE_TAG` 换回上一个版本再 `up -d`,切勿依赖 latest。
