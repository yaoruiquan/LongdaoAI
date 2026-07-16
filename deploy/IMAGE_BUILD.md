# 龙道 AI 生产镜像构建说明

对应 spec：REL-001（自有版本镜像构建）、REL-002（基于已提交 commit 构建）。

生产环境不再使用上游 `weishaw/sub2api:latest`，而是构建包含本地龙道 AI 改造
（前端品牌改版、迁移 136 等）的自有镜像，用**不可变版本号**部署，并保证镜像可
映射到唯一的 git commit，支持回滚到上一版本。

## 1. 版本 tag 规则

- 镜像全名：`${IMAGE_REPO}:<version>`，默认 `IMAGE_REPO=longdao/sub2api`。
- 生产版本号推荐日期化格式：`<YYYY.MM.DD>-<当日序号>`
  - 例：`2026.07.15-1`、`2026.07.15-2`
  - 优点：按时间排序、肉眼可读构建日期、天然不可变（同一 tag 不重复覆盖）。
- 每次构建同时打两个 tag：
  - 不可变：`longdao/sub2api:2026.07.15-1`（**生产 compose 必须引用此带版本 tag**）
  - 便利：`longdao/sub2api:latest`（仅本地/调试用，**禁止**用于生产回滚判断）。
- 版本号来源优先级（脚本内部）：
  1. 命令行第一个参数 或 环境变量 `VERSION`（显式，推荐）
  2. `git describe --tags --always --dirty`，退化为 `git rev-parse --short HEAD`
  3. 兜底 `backend/cmd/server/VERSION`（当前 `0.1.106`）

## 2. 如何构建

```bash
# 推荐：显式指定不可变版本号
./deploy/build_production_image.sh 2026.07.15-1

# 或用环境变量
VERSION=2026.07.15-1 ./deploy/build_production_image.sh

# 不传版本时用 git 自动生成（适合本地调试）
./deploy/build_production_image.sh

# 自定义镜像仓库前缀（如推送到私有 registry）
IMAGE_REPO=registry.example.com/longdao/sub2api \
  ./deploy/build_production_image.sh 2026.07.15-1
```

构建脚本会：

- 检查 git 工作区是否干净。若有未提交改动会**醒目警告并默认退出**（镜像无法精确
  映射到干净 commit，违反 REL-002）。仅调试可用 `ALLOW_DIRTY=1` 跳过。
- 采集 `COMMIT=$(git rev-parse HEAD)`（完整 SHA）与 UTC 构建时间 `DATE`，作为
  `--build-arg` 注入。Dockerfile 通过 ldflags 写入 `main.Version/Commit/Date`，
  并写入 OCI 标准 label（`org.opencontainers.image.revision/version/created`）。
- 保留国内构建加速参数 `GOPROXY=https://goproxy.cn,direct`、
  `GOSUMDB=sum.golang.google.cn`。

> 注意：`backend/cmd/server/VERSION` 保持 `0.1.106` 作为兜底，不随发布修改；实际
> 生产版本由脚本参数控制。

## 3. 验证镜像内容

### 3.1 版本/commit 映射（REL-001/REL-002）

```bash
# 查看 OCI label（应看到 revision=<完整 SHA>、version=<版本>、created=<UTC 时间>）
docker inspect --format '{{json .Config.Labels}}' longdao/sub2api:2026.07.15-1 | tr ',' '\n'

# 查看 digest（唯一标识镜像内容，用于推送/校验/审计）
docker images --digests longdao/sub2api

# 运行时确认二进制内注入的版本信息（/health 或版本接口，取决于应用暴露方式）
docker run --rm longdao/sub2api:2026.07.15-1 /app/sub2api --version 2>/dev/null || true
```

将 label 里的 `org.opencontainers.image.revision` 与 `git rev-parse HEAD` 对比，
即可确认镜像对应的确切 commit。

### 3.2 确认含龙道 AI 前端

前端已在构建阶段 `embed` 进二进制并打包进镜像。可启动后访问首页确认为龙道 AI 品牌
（页面标题/品牌名为“龙道 AI”），或用一次性容器抓取首页：

```bash
docker run --rm -p 18080:8080 longdao/sub2api:2026.07.15-1 &
sleep 5
curl -s http://localhost:18080/ | grep -i "龙道" && echo "前端品牌校验通过"
```

### 3.3 确认含迁移 136

迁移 `backend/migrations/136_restore_sora_schema_after_legacy_drop.sql` 通过 Go
`embed` 打包，随二进制在首次启动时自动执行。启动后可查看日志确认迁移 136 已应用：

```bash
docker compose logs sub2api | grep -i "136" || docker logs sub2api | grep -i "136"
```

## 4. 生产部署 / compose image 字段

当前 `deploy/docker-compose.yml` 的 `sub2api.image` 仍为上游：

```yaml
services:
  sub2api:
    image: weishaw/sub2api:latest   # 需改为龙道 AI 带版本 tag
```

生产部署应改为带版本的不可变 tag（此项属 DEP-001，本次不改文件，仅说明）：

```yaml
services:
  sub2api:
    image: longdao/sub2api:2026.07.15-1
```

## 5. 回滚到上一镜像版本

回滚即“把 compose 的 image 换回上一个版本 tag”：

1. 编辑 `deploy/docker-compose.yml`，把 `sub2api.image` 改回上一个版本，例如
   `longdao/sub2api:2026.07.14-1`。
2. 执行 `docker compose up -d` 重新拉起。

关键约束：

- 生产**只用带版本 tag**，不要用 `latest` 判断当前/回滚目标。
- 保留最近若干个版本镜像（不要 `docker image prune` 掉旧版本），以便快速回滚。
- 每个版本 tag 不可变：修复问题请构建新版本号，切勿覆盖已发布的 tag。
