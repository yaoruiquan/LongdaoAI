# 龙道 AI 标准发布与回滚流程

> 这份文档解释"为什么这样做"。实际操作命令见 `deploy/production/OPS_README.md`，脚本在 `deploy/production/` 目录下。

---

## 1. 发布流程的黄金准则

在进入具体步骤之前，先理解四条不可违反的原则。这些原则来自于真实事故的教训，不是学术建议。

**原则一：数据库备份必须先于任何迁移**

迁移一旦执行就无法自动撤销（尤其是删列、改类型这类破坏性变更）。如果迁移中途失败或新版本有问题，你唯一的退路就是备份。备份不在手，回滚等于赌博。

**原则二：迁移成功才能启动新版本**

新版代码依赖新的数据库结构。如果迁移失败但新容器已经启动，应用会因为找不到新字段/表而报错，且可能还破坏了旧结构导致旧版本也跑不起来。正确做法：迁移失败就停，不启动新版本。

**原则三：镜像版本必须不可变，禁止 `latest`**

`latest` 是可变指针，你不知道今天和昨天的 `latest` 是不是同一个镜像。用 `v2026.07.17-1` 这样的日期版本号，每次发布都有唯一标识，可以精确回滚到任意历史版本。

**原则四：每次发布都需要健康检查和冒烟测试**

部署完成不等于功能正常。健康检查只能告诉你"进程活着"，冒烟测试才能确认"核心功能可用"。任何一步失败，必须在通知用户之前先处理好。

---

## 2. 标准发布序列

```
备份 → 构建新镜像 → 更新 IMAGE_TAG → 跑迁移 → up -d → 健康检查 → 冒烟测试
```

### 步骤详解

#### 步骤 0：构建不可变版本镜像

```bash
# 在仓库根目录执行，基于当前已提交的代码打包
./deploy/build_production_image.sh 2026.07.17-1
```

**为什么**：镜像必须基于已提交的代码，不能包含未提交的修改，确保版本号和代码状态一一对应。构建完成后镜像就"冻结"了，后续发布用的就是这个镜像。

**失败处理**：检查 Dockerfile、Go 编译错误、网络（拉取基础镜像）。修复后重新构建，版本号可以加后缀（如 `2026.07.17-2`）。

#### 步骤 1：更新 .env 中的 IMAGE_TAG

```bash
# 编辑 deploy/production/.env
IMAGE_TAG=v2026.07.17-1
```

**为什么**：compose 文件通过 `${IMAGE_TAG:?...}` 读取版本号。手动更新 `.env` 而不是修改 compose 文件，是为了让 compose 文件保持在代码仓库中可追踪，而实际部署参数（哪个版本）留在不进仓库的 `.env` 里。

#### 步骤 2：发布前备份

```bash
./deploy/production/backup.sh
```

产物举例：
- `pg_prod_v2026.07.17-1_20260717T030000.sql.gz`（PostgreSQL dump，压缩后通常几十 MB）
- `data_prod_v2026.07.17-1_20260717T030000.tar.gz`（`/app/data` 目录，含 config.yaml）

**为什么**：这是迁移前的最后一个已知好状态。如果本次发布包含破坏性迁移（删列等），回滚时需要从这个备份恢复数据库。

#### 步骤 3：执行数据库迁移

```bash
./deploy/production/migrate.sh
# 内部等价于：
# docker compose run --rm sub2api -migrate
```

`-migrate` 是应用自带的迁移模式，它只跑迁移然后退出，不启动 HTTP 服务。迁移成功返回退出码 0，失败返回非零（`deploy.sh` 据此中止流程）。

**失败处理**：不要继续启动新版本。查看迁移日志定位失败的 SQL 文件，修复后重新执行。如果迁移已经部分应用且无法前进，需要从备份恢复数据库（见第 4 节）。

#### 步骤 4：启动新版本

```bash
docker compose -f deploy/production/docker-compose.yml --env-file deploy/production/.env up -d
```

Docker 会：
1. 比较当前运行的镜像和 `.env` 里指定的镜像
2. 只重建有变化的服务（通常只有 `sub2api`）
3. postgres/redis/caddy 正在运行且没有配置变化时不重启

#### 步骤 5：健康检查

```bash
# 等待 sub2api 容器变为 healthy（最多等 90 秒）
docker compose -f deploy/production/docker-compose.yml ps
```

`sub2api` 的 healthcheck 配置了 `start_period: 30s`，意味着启动后 30 秒内的失败不计入。如果超过 `30s + retries*interval = 30+90=120s` 还是 unhealthy，认为启动失败。

#### 步骤 6：冒烟测试

```bash
./deploy/production/smoke-test.sh
# 或指定域名：
BASE_URL=https://longdaoai.cn ./deploy/production/smoke-test.sh
```

P0 项：`/health` 返回 200，失败即退出码非零，`deploy.sh` 据此判定发布失败。

**整个流程由 `deploy.sh` 自动编排**，任意步骤失败就停下来并提示回滚：

```bash
./deploy/production/deploy.sh
```

---

## 3. 数据库迁移的关键知识

### 3.1 迁移文件的幂等性（IF NOT EXISTS）

本项目迁移文件（`backend/migrations/NNN_描述.sql`）中大量使用幂等写法：

```sql
-- 幂等：表已存在就跳过，不报错
CREATE TABLE IF NOT EXISTS users (...);

-- 幂等：列已存在就跳过
ALTER TABLE channels ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

-- 幂等：索引已存在就跳过
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
```

**为什么重要**：在多人协作或分支合并时，可能出现同一个迁移被执行两次的情况（比如备份恢复后重跑迁移）。幂等迁移保证重复执行没有副作用。

### 3.2 schema_migrations 表

应用在数据库里维护一张 `schema_migrations` 表，按迁移文件名记录哪些迁移已经执行过：

```
schema_migrations
──────────────────────────────────
filename                  | checksum
134_affiliate_ledger...   | abc123...
135_xxx...                | def456...
181_yyy...                | ghi789...
```

每次跑迁移时，应用会扫描 `backend/migrations/` 目录，跳过已记录的文件，只执行新的。

### 3.3 checksum 校验

`schema_migrations` 表还存储了每个迁移文件的 checksum。如果一个已执行的迁移文件被修改，下次迁移时 checksum 不匹配，应用会拒绝执行并报错。

**这是一个保护机制**，防止有人"悄悄修改"已应用的迁移文件（这样做会导致数据库实际状态和迁移文件不一致，非常危险）。

如果确实需要修复一个有问题的已应用迁移，正确做法是**新建一个更高序号的迁移文件**来修正，而不是修改原文件。

### 3.4 fork 分叉导致的迁移序列冲突（本次实际遇到的情况）

本项目在从上游仓库升级时遇到了一个典型问题：

**背景**：本项目基于上游 sub2api 开发，本地添加了自定义迁移（如 `136_custom_feature.sql`）。上游在 v2 版本也新增了迁移，序号同样从 136 开始，导致序号冲突。

**症状**：
- 上游迁移文件 `136_upstream_feature.sql` 与本地 `136_custom_feature.sql` 序号相同
- 如果数据库已执行了本地的 136，上游的 136 会因为 checksum 不匹配被拒绝
- 直接 `docker compose run --rm sub2api -migrate` 失败

**解法：把本地迁移重命名为更高序号**

```bash
# 本地迁移 136 与上游冲突，重命名为不会冲突的高序号
git mv backend/migrations/136_custom_feature.sql \
       backend/migrations/181_custom_feature.sql
```

选择 181 的依据：检查上游最新迁移文件的最高序号（当时是 180），取 181 以上。

**幂等迁移如何化解冲突**：因为本地 136 里的 SQL 都写了 `IF NOT EXISTS`，重命名后以 181 号重新执行，实际上不会重复建表/列，只是补记到 `schema_migrations` 表。如果迁移不幂等，这个操作就会报错。

---

## 4. 回滚策略

### 4.1 仅回滚应用（不涉及破坏性迁移的情况）

95% 的情况只需要：

```bash
./deploy/production/rollback.sh v2026.07.15-1
```

内部等价于：
```bash
IMAGE_TAG=v2026.07.15-1 docker compose ... up -d sub2api
```

这只替换 `sub2api` 容器的镜像，**postgres/redis 数据卷完全不动**。回滚完成后自动健康检查。

完成后把 `.env` 里的 `IMAGE_TAG` 也改回旧版本（否则下次执行 `deploy.sh` 会再次变成新版本）。

### 4.2 何时需要恢复数据库备份

如果本次发布包含**不兼容迁移**（以下任意一种）：

- 删除了列（`DROP COLUMN`）
- 修改了列类型（`ALTER COLUMN ... TYPE`）
- 删除了表
- 修改了约束导致旧数据无效

那么仅回滚应用镜像是不够的——旧版代码期望的列/表已经不存在了，应用会报错。这时需要：

```bash
# 1. 先把应用回滚（阻止新数据写入）
./deploy/production/rollback.sh v2026.07.15-1

# 2. 停应用（避免回滚数据库时有新写入）
docker compose stop sub2api

# 3. 从发布前备份恢复数据库（高危操作，需二次确认）
./deploy/production/restore.sh deploy/production/backups/pg_prod_v2026.07.17-1_xxx.sql.gz sub2api

# 4. 重启应用
docker compose start sub2api
```

### 4.3 "前向修复"原则

**优先修代码，而非回退数据库。**

数据库回滚代价高昂（停服、丢失这段时间的新数据、需要备份可用）。如果 bug 是代码逻辑问题而非数据结构问题，更好的做法是：

1. 快速修复代码 bug
2. 构建新镜像（如 `v2026.07.17-2`）
3. 按标准发布流程发布修复版本

只有数据库结构本身有问题（比如添加了 NOT NULL 约束导致旧数据插入失败），才考虑数据库回滚。

---

## 5. 升级到新版本（大版本迁移）

### 5.1 本次经历：2355 commits 的跨版本升级

本项目在 2026-07-16 完成了从上游旧分支到 v2 的升级，上游有约 2355 个新 commit，横跨多个 feature。

两种策略对比：

| 策略 | 适用场景 | 风险 |
|------|---------|------|
| `git rebase` 上游分支 | 本地改动少（<10 个 commit），改动局限于配置/小功能 | 大量冲突，容易出错，历史混乱 |
| 全新起步 + 移植定制 | 本地深度定制（数据库结构、业务逻辑），上游变化巨大 | 需要手动整理差异，但结果干净 |

**本次选择了"全新起步+移植定制"**：

1. 以上游 v2 最新代码为基础创建新分支
2. 逐一识别本地定制（迁移文件、Dockerfile、compose、脚本）
3. 把定制内容移植到新基础上（迁移文件重命名到高序号避免冲突）
4. 重新验证构建和部署

### 5.2 何时选 rebase，何时选重新移植

**选 rebase 的条件**（同时满足）：
- 本地改动少于 20 个 commit
- 改动与上游改动几乎没有交叉（改了不同文件）
- 上游变化不超过 200 个 commit

**选重新移植的条件**（满足任意一条）：
- 上游有大规模重构（目录结构变了、核心接口改了）
- 本地改动的文件与上游大量重叠
- 想要一个干净的历史

### 5.3 集成测试需要 DB 的问题

本次升级后发现 `go test ./...` 在无数据库的环境下失败，错误类似：

```
FAIL  github.com/xxx/sub2api/internal/repository  [setup failed]
FAIL  github.com/xxx/sub2api/backend/ent/schema   [setup failed]
```

原因：`ent/schema` 包的测试会尝试连接数据库做 schema 验证；`internal/repository` 的集成测试同样需要 DB。这不是代码 bug，而是测试分类问题。

**处理方式**：
- CI/CD 流水线需要提供数据库服务（如 GitHub Actions 的 `services.postgres`）
- 本地跑测试前先 `docker compose up -d postgres`
- 或者只跑单元测试（用 `-run` 指定或用 build tag 隔离集成测试）

---

## 附：快速参考卡

### 正常发布

```bash
# 1. 构建新镜像
./deploy/build_production_image.sh 2026.07.17-1

# 2. 更新 .env 里的 IMAGE_TAG=v2026.07.17-1

# 3. 一键发布（含备份+迁移+健康检查+冒烟）
./deploy/production/deploy.sh
```

### 快速回滚

```bash
./deploy/production/rollback.sh v2026.07.15-1
# 回滚后记得手动改 .env 里的 IMAGE_TAG
```

### 手动备份

```bash
./deploy/production/backup.sh
```

### 查看发布日志

```bash
tail -50 deploy/production/deploy.log
```

---

最后更新：2026-07-17
