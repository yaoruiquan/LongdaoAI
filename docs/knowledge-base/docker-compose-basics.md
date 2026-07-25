# Docker 与 Docker Compose 核心概念

> 面向有一定开发基础、但没深入用过容器的开发者。读完本文你应该能看懂本项目的 compose 文件，知道每条配置"为什么在这里"。

---

## 1. 镜像 vs 容器

**镜像（Image）** 是一个只读的"安装包"。它把应用代码、依赖库、运行时环境全部打包进去，放在那里不动。

**容器（Container）** 是镜像"跑起来"之后的实例。同一个镜像可以同时启动多个容器，就像同一个 `.app` 文件可以打开多个窗口。

```
镜像                容器（运行中）
──────────         ─────────────────
longdao/sub2api  → longdao-sub2api（正在处理请求）
:v2026.07.17-1   → longdao-sub2api-2（蓝绿部署时的新实例）
```

容器是临时的。删除容器不会删除镜像，也不会删除挂载的数据卷（见第 4.2 节）。

---

## 2. Docker Compose 是什么

手写 `docker run` 命令又长又难维护。Docker Compose 把多个容器的配置集中写在一个 `docker-compose.yml` 文件里，然后用一条命令管理整个应用栈：

```bash
docker compose up -d      # 启动所有服务
docker compose down       # 停止并删除所有容器
```

本项目的生产配置在 `deploy/production/docker-compose.yml`，用它一次性管理四个服务。

---

## 3. 本项目的四服务架构

```
公网请求
    │
    ▼
┌─────────────┐  443/80
│    caddy    │  ← 唯一对外暴露端口，自动签发 HTTPS 证书
└──────┬──────┘
       │ 内部网络 longdao-network
       ▼
┌─────────────┐  :8080（仅内部可达）
│   sub2api   │  ← Go 应用，API 网关 + 业务逻辑
└──────┬──────┘
       │
   ┌───┴───┐
   ▼       ▼
┌──────┐ ┌──────┐
│  pg  │ │redis │  ← 不暴露任何端口到宿主机
└──────┘ └──────┘
  数据库    缓存/限流
```

| 服务 | 镜像 | 作用 |
|------|------|------|
| `sub2api` | `longdao/sub2api:v…` | 核心业务：用户管理、Token 中转、计费统计 |
| `postgres` | `postgres:18-alpine` | 主数据库：用户、渠道、账单等持久数据 |
| `redis` | `redis:8-alpine` | 缓存 + 速率限制（RPM/TPM 计数器） |
| `caddy` | `caddy:2-alpine` | 反向代理 + 自动 HTTPS（Let's Encrypt） |

---

## 4. 关键概念详解

### 4.1 网络（network）

Compose 为服务创建一个内部虚拟网络（本项目名为 `longdao-network`）。同一网络内的容器可以直接用**服务名**当域名互相访问：

```yaml
# sub2api 的环境变量
DATABASE_HOST=postgres   # 不是 IP，是服务名
REDIS_HOST=redis
```

这样 `sub2api` 访问 `postgres:5432` 就能直接连到 postgres 容器，不需要知道 IP。

**为什么 postgres/redis 不暴露公网端口？**

生产 compose 里 postgres 和 redis 都没有 `ports` 配置。它们只在 `longdao-network` 内可达，宿主机外部（包括公网）完全访问不到。这是纵深防御的基本做法——数据库不应该直接暴露在公网，即使有密码保护也会增大攻击面。

需要调试时可以临时加（注意绑定到 `127.0.0.1` 而不是 `0.0.0.0`）：
```yaml
ports:
  - "127.0.0.1:5433:5432"   # 只有本机能访问，公网不行
```

### 4.2 数据卷（volume）

容器本身是无状态的，一旦删除，容器内写入的文件全部消失。数据卷把容器内的目录"挂载"到宿主机管理的存储空间，容器重建后数据依然在。

```yaml
volumes:
  - postgres_data:/var/lib/postgresql/data   # 命名卷
```

本项目的命名卷：

| 卷名 | 挂载点 | 存放什么 |
|------|--------|---------|
| `longdao_postgres_data` | `/var/lib/postgresql/data` | 所有数据库数据 |
| `longdao_redis_data` | `/data` | Redis AOF 持久化文件 |
| `longdao_sub2api_data` | `/app/data` | config.yaml 等应用配置 |
| `longdao_caddy_data` | `/data` | ACME 证书与 Let's Encrypt 账户 |

**`docker compose down -v` 的危险性**

普通的 `docker compose down` 只删容器，数据卷完好无损。但加上 `-v` 标志会**一并删除所有命名卷**，数据库数据、证书全部销毁，且无法恢复（除非有备份）。

```bash
docker compose down        # 安全：只删容器
docker compose down -v     # 危险：连数据一起删，生产永远不要用
```

### 4.3 端口映射

格式：`"宿主机地址:宿主机端口:容器端口"`

```yaml
# 开发环境：绑定到 0.0.0.0，所有网卡都能访问
ports:
  - "0.0.0.0:8080:8080"   # 或简写 "8080:8080"

# 生产/调试：只绑定到回环地址，只有本机能访问
ports:
  - "127.0.0.1:8080:8080"
```

**安全区别**：绑定 `0.0.0.0` 意味着公网 IP 也在监听，任何人都能访问这个端口。绑定 `127.0.0.1` 则只有 SSH 登录到服务器后才能访问，公网无法直连。

生产环境里 `sub2api` 完全不暴露端口（caddy 在内部网络内直接访问 `sub2api:8080`），caddy 只暴露 80/443 并绑定到 `0.0.0.0`——这是正确的做法。

### 4.4 健康检查（healthcheck）

容器"运行"不代表"就绪"。应用可能正在初始化、数据库可能还没接受连接。健康检查让 Docker 知道服务真正可用：

```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U sub2api -d sub2api"]
  interval: 10s      # 每 10 秒检查一次
  timeout: 5s        # 超时判为失败
  retries: 5         # 连续 5 次失败才标记 unhealthy
  start_period: 10s  # 启动后等 10 秒再开始检查（给应用初始化时间）
```

`start_period` 的作用：PostgreSQL 启动时需要初始化数据目录，这段时间内检查必然失败。`start_period` 告诉 Docker"这段时间内的失败不算数"，避免还没启动好就被标为 unhealthy。

`depends_on: condition: service_healthy` 的意义：

```yaml
# sub2api 不会启动，直到 postgres 和 redis 都报告 healthy
depends_on:
  postgres:
    condition: service_healthy
  redis:
    condition: service_healthy
```

如果没有这个条件，sub2api 可能在 postgres 还没就绪时就尝试连接数据库，导致启动失败。

### 4.5 资源限制

```yaml
deploy:
  resources:
    limits:
      cpus: "2.0"    # 最多使用 2 个核心
      memory: 1536M  # 最多使用 1.5 GB 内存
    reservations:
      cpus: "0.5"    # 软预留（调度参考）
      memory: 512M
```

没有 `limits` 时，一个容器可以吃掉全部内存，导致其他容器被系统 OOM Kill。本项目各服务都设了 limits，防止单个服务资源失控影响整体。

### 4.6 日志限制

容器日志默认会一直写到磁盘满为止。加上日志轮转配置：

```yaml
logging:
  driver: json-file
  options:
    max-size: "20m"   # 单个日志文件最大 20MB
    max-file: "5"     # 保留最近 5 个文件，最多占用 100MB
```

生产环境四个服务都加了这个配置，日志最多占用 400MB，不会把磁盘打满。

---

## 5. 常用命令

| 命令 | 场景 |
|------|------|
| `docker compose up -d` | 后台启动所有服务（已运行的服务不重启） |
| `docker compose up -d sub2api` | 只重启 sub2api（其他服务不动） |
| `docker compose down` | 停止并删除所有容器（数据卷保留） |
| `docker compose ps` | 查看各服务状态和健康情况 |
| `docker compose logs -f sub2api` | 实时跟踪 sub2api 的日志（Ctrl+C 退出） |
| `docker compose logs --tail=100 caddy` | 查看 caddy 最近 100 行日志 |
| `docker compose exec -T sub2api sh` | 进入运行中的 sub2api 容器 |
| `docker compose run --rm sub2api -migrate` | 一次性容器跑数据库迁移（用完即删） |
| `docker compose config -q` | 校验 compose 文件语法（检查变量替换） |

`docker compose run --rm` 和 `docker compose exec` 的区别：
- `exec` 是在**已运行的容器**里执行命令，容器必须先 up。
- `run --rm` 是**新建一个临时容器**，命令跑完容器自动删除。跑数据库迁移通常用这个，因为不想让迁移逻辑干扰正在运行的应用容器。

---

## 6. 镜像版本管理

本项目生产环境**禁止使用 `latest` 标签**，使用日期版本号：

```
longdao/sub2api:v2026.07.17-1
```

**为什么不用 `latest`：**
- `latest` 是可变指针，今天和明天拉到的可能是不同的镜像，无法复现问题。
- 无法回滚到"上一个版本"，因为 latest 随时被覆盖。
- 部署时无法确认当前运行的是哪个构建。

**如何回滚：**

```bash
# 在 .env 中改回上一个版本号
IMAGE_TAG=v2026.07.15-1

# 重新 up，Docker 会换用旧镜像重建容器，数据卷不动
docker compose -f deploy/production/docker-compose.yml --env-file deploy/production/.env up -d sub2api
```

回滚只需改 `IMAGE_TAG`，不需要动数据库，秒级完成。

---

## 7. 本项目的 `.env` 机制

所有密钥和环境相关配置都放在 `deploy/production/.env`，该文件在 `.gitignore` 中，**不进代码仓库**。

```bash
# .env 示例（真实值不提交 git）
POSTGRES_PASSWORD=强密码...
REDIS_PASSWORD=强密码...
JWT_SECRET=openssl rand -hex 32 的输出
TOTP_ENCRYPTION_KEY=openssl rand -hex 32 的输出
IMAGE_TAG=v2026.07.17-1
LONGDAO_DOMAIN=longdaoai.cn
```

Compose 文件里用 `${变量名:?错误提示}` 语法引用，如果变量未设置，`docker compose up` 会直接报错而不是用空值启动：

```yaml
image: longdao/sub2api:${IMAGE_TAG:?IMAGE_TAG is required}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
```

**为什么密钥不能进 git：** 一旦提交到远程仓库（即使是私有仓库），密钥就可能被协作者、CI/CD 系统、以及历史记录永久保存。泄露的密钥必须立即轮换，代价很高。

---

## 8. 常见问题

### Q：caddy 容器显示 unhealthy，但网站能正常访问

本项目 caddy 健康检查探的是本地 Admin API：

```yaml
test: ["CMD", "wget", "-q", "-T", "5", "-O", "/dev/null", "http://127.0.0.1:2019/config/"]
```

如果服务器关闭了 IPv6 或本地回环有特殊配置，这个检查可能误报。实际上只要 `https://longdaoai.cn` 能正常访问，caddy 就是工作正常的——`unhealthy` 是健康检查配置与实际环境不匹配，不代表服务本身挂掉。

### Q：`go test ./...` 在 CI 或本地失败，提示 setup failed

本项目的集成测试（`ent/schema`、`internal/repository` 等包）需要连接真实数据库才能初始化。在没有数据库的纯 CI 环境里这些测试无法运行。

区别：
- **单元测试**：只测逻辑，mock 掉外部依赖，无需 DB。
- **集成测试**：需要真实 DB，必须先 `docker compose up postgres`。

### Q：构建镜像时间很长

Dockerfile 使用了 `--mount=type=cache` 来缓存 Go 模块依赖：

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build ...
```

首次构建时需要下载所有依赖，后续构建如果依赖没变化会命中缓存，速度快很多。在全新服务器（缓存为空）或 `docker buildx` 没有持久化缓存时，首次构建慢是正常的。

---

最后更新：2026-07-17
