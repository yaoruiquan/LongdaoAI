# 账号自动恢复功能 - 快速部署指南

## 本次更新内容

1. **新增功能**：`AccountRecoveryService` - 每 3 小时自动恢复过期的临时禁用账号
2. **故障排查工具**：
   - `deploy/production/troubleshoot-503.sh` - 一键诊断脚本
   - `deploy/production/TROUBLESHOOT.md` - 完整故障排查指南
3. **已修复问题**：给 `gpt优惠` 分组添加了 5 个备份账号

## 快速部署（推荐）

### 方式 1：直接在生产服务器上构建（推荐）

这种方式避免了本地网络问题，在服务器上直接编译和部署。

```bash
# 1. 提交代码到仓库
cd /Users/yao/LLM/sub2api
git add backend/internal/service/account_recovery.go \
        backend/internal/service/account_recovery_test.go \
        backend/internal/service/wire.go \
        backend/cmd/server/wire.go \
        deploy/production/troubleshoot-503.sh \
        deploy/production/TROUBLESHOOT.md \
        docs/features/account-recovery.md
        
git commit -m "feat(service): 添加账号自动恢复功能

- 新增 AccountRecoveryService，每 3 小时自动恢复过期的临时禁用账号
- 解决账号临时失效后无法自动恢复导致的 503 故障
- 添加故障排查工具和文档
- 修复：给 gpt优惠 分组添加 5 个备份账号

Co-Authored-By: Claude <noreply@anthropic.com>"

git push

# 2. 登录服务器拉取代码
ssh root@longdaoai.cn
cd /opt/longdao
git pull

# 3. 生成 wire 依赖注入代码
cd backend
go generate ./cmd/server

# 4. 构建镜像
cd /opt/longdao
./deploy/build_production_image.sh 2026.07.29-1

# 5. 更新 IMAGE_TAG 并发布
vim deploy/production/.env
# 设置：IMAGE_TAG=v2026.07.29-1

./deploy/production/deploy.sh

# 6. 验证部署
docker logs longdao-sub2api-blue --tail 50 | grep "account_recovery"
# 应该看到：account recovery service started interval=3h0m0s
```

### 方式 2：本地构建（如果网络正常）

```bash
cd /Users/yao/LLM/sub2api

# 1. 生成 wire 代码
cd backend
go generate ./cmd/server

# 2. 运行测试
go test -v -run TestAccountRecoveryService ./internal/service/

# 3. 构建镜像
cd ..
./deploy/build_production_image.sh 2026.07.29-1

# 4. 推送到服务器（如果有私有仓库）
docker push longdao/sub2api:2026.07.29-1

# 5. SSH 到服务器部署
ssh root@longdaoai.cn
cd /opt/longdao
vim deploy/production/.env
# 设置：IMAGE_TAG=v2026.07.29-1
./deploy/production/deploy.sh
```

## 验证功能

### 1. 查看服务启动日志

```bash
ssh root@longdaoai.cn
docker logs longdao-sub2api-blue --tail 100 | grep account_recovery
```

期望输出：
```
INFO account recovery service started interval=3h0m0s
```

### 2. 手动触发恢复（测试用）

如果想立即测试恢复功能，可以手动创建一个过期的临时禁用账号：

```sql
-- 在生产数据库执行
UPDATE accounts 
SET status = 'error',
    schedulable = false,
    temp_unschedulable_until = NOW() - INTERVAL '1 hour',
    temp_unschedulable_reason = 'Test recovery'
WHERE id = <某个测试账号ID>;

-- 等待最多 3 小时后，账号应该自动恢复
-- 或者重启服务立即触发一次恢复
```

### 3. 监控恢复记录

```sql
-- 查询最近恢复的账号（通过 updated_at 判断）
SELECT 
  id, 
  name, 
  status, 
  schedulable, 
  updated_at
FROM accounts 
WHERE deleted_at IS NULL
  AND status = 'active'
  AND schedulable = true
  AND updated_at > NOW() - INTERVAL '4 hours'
ORDER BY updated_at DESC;
```

## 回滚方案

如果新版本有问题，执行：

```bash
ssh root@longdaoai.cn
cd /opt/longdao
./deploy/production/rollback.sh v2026.07.26-1  # 回滚到上一个版本
```

## 注意事项

1. **首次运行会立即执行一次恢复**，检查所有过期的临时禁用账号
2. **之后每 3 小时定期执行**
3. **日志级别**：如果没有需要恢复的账号，只会记录 `DEBUG` 级别日志
4. **数据库账号已修复**：`gpt优惠` 分组现在有 6 个可用账号，单点故障已解决

## 相关文档

- 完整功能说明：`docs/features/account-recovery.md`
- 故障排查指南：`deploy/production/TROUBLESHOOT.md`
- 部署流程说明：`docs/knowledge-base/deployment-flow.md`
