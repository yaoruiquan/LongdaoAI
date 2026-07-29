# 账号自动恢复功能

## 功能说明

自动恢复已过期的临时禁用账号，避免因账号临时失效导致的服务中断。

## 实现细节

### 服务：`AccountRecoveryService`

- **检查间隔**：每 3 小时执行一次
- **启动时机**：应用启动时自动运行一次，然后每 3 小时定期执行
- **恢复条件**：
  1. `deleted_at IS NULL`（未删除）
  2. `schedulable = false`（当前不可调度）
  3. `temp_unschedulable_until IS NOT NULL`（有设置临时禁用时间）
  4. `temp_unschedulable_until < NOW()`（禁用时间已过期）

### 恢复操作

当满足以上条件时，自动执行：
```sql
UPDATE accounts SET
  status = 'active',
  schedulable = true,
  error_message = NULL,
  temp_unschedulable_until = NULL,
  temp_unschedulable_reason = NULL
WHERE <上述条件>
```

## 部署说明

### 1. 本地测试

```bash
# 运行测试
cd backend
go test -v -run TestAccountRecoveryService ./internal/service/

# 重新生成 wire 依赖注入代码
go generate ./cmd/server

# 编译
go build -o sub2api ./cmd/server
```

### 2. 构建镜像

```bash
# 在仓库根目录执行
./deploy/build_production_image.sh 2026.07.29-1
```

### 3. 部署到生产

```bash
# 1. 更新 .env 中的 IMAGE_TAG
ssh root@longdaoai.cn
cd /opt/longdao
vim deploy/production/.env
# 设置：IMAGE_TAG=v2026.07.29-1

# 2. 执行发布（包含备份+迁移+蓝绿切换）
./deploy/production/deploy.sh
```

### 4. 验证部署

```bash
# 查看日志，确认服务启动
docker logs longdao-sub2api-blue --tail 100 | grep "account_recovery"

# 应该看到类似日志：
# account recovery service started interval=3h0m0s
```

## 监控建议

### 手动触发恢复（调试用）

如果需要手动触发账号恢复，可以直接在数据库执行：

```sql
UPDATE accounts 
SET status = 'active', 
    schedulable = true,
    error_message = NULL,
    temp_unschedulable_until = NULL,
    temp_unschedulable_reason = NULL
WHERE deleted_at IS NULL
  AND schedulable = false
  AND temp_unschedulable_until IS NOT NULL
  AND temp_unschedulable_until < NOW();
```

### 查询待恢复账号

```sql
SELECT 
  id, 
  name, 
  platform, 
  status, 
  temp_unschedulable_until,
  temp_unschedulable_reason,
  NOW() - temp_unschedulable_until as expired_duration
FROM accounts 
WHERE deleted_at IS NULL
  AND schedulable = false
  AND temp_unschedulable_until IS NOT NULL
  AND temp_unschedulable_until < NOW();
```

### 告警配置（推荐）

建议配置以下告警：
1. 当某个分组的可用账号数 < 2 时，发送告警
2. 当账号状态变为 `error` 时，记录到日志
3. 每次自动恢复账号时，记录恢复数量

## 历史问题回顾

### 2026-07-29 故障

**现象**：
- 所有用户无法访问（503 错误）
- 前端管理后台也无法打开

**根本原因**：
- `gpt优惠` 分组（用户最多的分组）只绑定了 1 个账号（账号 4）
- 该账号在 7月25日 因上游 403 错误被标记为 `status=error`, `schedulable=false`
- 系统没有自动恢复机制，导致即使 `temp_unschedulable_until` 过期，账号状态也没有恢复

**临时修复**：
1. 手动恢复账号 4 的状态
2. 给 `gpt优惠` 分组添加 5 个备份账号（GLM、deepseek、kimi、aipro codex、硅基流动）

**长期解决方案**：
1. ✅ 实现自动恢复机制（本次更新）
2. ✅ 每个分组至少绑定 3 个账号（已修复）
3. 🔄 配置账号健康检查（待实现）
4. 🔄 配置监控告警（待实现）

## 相关文件

- 服务实现：`backend/internal/service/account_recovery.go`
- 测试文件：`backend/internal/service/account_recovery_test.go`
- 依赖注入：`backend/internal/service/wire.go`
- 清理逻辑：`backend/cmd/server/wire.go`
- 故障排查脚本：`deploy/production/troubleshoot-503.sh`
- 故障排查指南：`deploy/production/TROUBLESHOOT.md`
