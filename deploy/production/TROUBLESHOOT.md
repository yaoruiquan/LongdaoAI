# 503 故障排查指南

## 问题现象

- **所有用户**同时无法访问（非个别用户问题）
- 前端显示 "Request failed with status code 503"
- 管理后台界面报错，接口全部 503
- **过一会自动恢复**（说明不是配置错误，是暂时性故障）

---

## 快速排查（推荐）

在生产服务器上运行一键排查脚本：

```bash
# 方式 1：直接在服务器上执行
ssh root@longdaoai.cn
cd /root/sub2api  # 或你的实际部署路径
bash deploy/production/troubleshoot-503.sh

# 方式 2：从本地推送并执行（如果服务器没有最新脚本）
scp deploy/production/troubleshoot-503.sh root@longdaoai.cn:/root/sub2api/deploy/production/
ssh root@longdaoai.cn 'cd /root/sub2api && bash deploy/production/troubleshoot-503.sh'
```

脚本会自动检查：
1. 容器运行状态
2. 健康检查状态
3. 后端日志（最近 100 行）
4. Caddy 反向代理日志
5. 数据库/Redis 连接
6. 系统资源（内存/磁盘/CPU）
7. 内部网络连通性
8. 容器重启记录

---

## 常见原因及解决方案

### 1. 后端容器健康检查失败

**症状**：`docker ps` 显示后端容器状态为 `unhealthy`

**原因**：
- `/health` 端点响应超时（默认 3 秒超时）
- 数据库连接池耗尽
- 内存不足导致 Go 进程响应缓慢

**解决**：
```bash
# 查看健康检查日志
docker inspect --format='{{json .State.Health}}' sub2api-backend | jq

# 重启后端容器
docker compose -f deploy/production/docker-compose.yml restart sub2api-backend

# 如果持续 unhealthy，检查后端日志
docker logs sub2api-backend --tail 200
```

---

### 2. 上游账号全部失效

**症状**：
- 后端日志大量 "upstream API error"
- 所有转发请求失败
- 过一会恢复（上游 API 限流解除）

**解决**：
```bash
# 检查账号健康状态（在管理后台）
# 访问：https://longdaoai.cn/admin/accounts

# 临时措施：手动禁用失效账号，只保留健康账号
# 长期方案：配置账号自动探活和熔断
```

---

### 3. 数据库连接耗尽

**症状**：
- 后端日志显示 "pq: sorry, too many clients already"
- 高并发时触发

**解决**：
```bash
# 检查当前连接数
docker exec sub2api-postgres psql -U sub2api -d sub2api -c \
  "SELECT count(*) FROM pg_stat_activity WHERE datname='sub2api';"

# 查看最大连接数配置
docker exec sub2api-postgres psql -U sub2api -c "SHOW max_connections;"

# 临时增加连接数（需重启 postgres）
# 编辑 deploy/production/docker-compose.yml，在 postgres 服务添加：
# command: postgres -c max_connections=200

# 长期方案：优化应用层连接池配置
# 在 .env 中设置：
# DB_MAX_OPEN_CONNS=50
# DB_MAX_IDLE_CONNS=10
```

---

### 4. 内存不足 / OOM Killer

**症状**：
- 容器频繁重启
- `dmesg` 显示 OOM killed
- 系统内存接近 100%

**解决**：
```bash
# 检查系统内存
free -h

# 检查容器内存限制
docker stats --no-stream

# 检查是否被 OOM killed
dmesg | grep -i "killed process"

# 临时措施：重启容器
docker compose -f deploy/production/docker-compose.yml restart

# 长期方案：
# 1. 升级服务器内存
# 2. 优化代码（检查内存泄漏）
# 3. 在 docker-compose.yml 中设置合理的内存限制
```

---

### 5. 蓝绿部署切换时的短暂中断

**症状**：
- 发生在发布/回滚期间
- 持续时间 < 10 秒
- 自动恢复

**原因**：健康检查通过后，Caddy 需要几秒钟更新上游列表（reload 期间）

**解决**：
- 这是正常现象，无需处理
- 如果想完全消除，需要改用更复杂的负载均衡方案（如 Nginx + Consul）

---

### 6. Caddy 反向代理配置错误

**症状**：
- 持续 503，不会自动恢复
- Caddy 日志显示 "no healthy upstream"

**解决**：
```bash
# 检查 Caddyfile 配置
docker exec sub2api-caddy cat /etc/caddy/Caddyfile

# 检查 Caddy 能否连接后端
docker exec sub2api-caddy wget -q -O- http://sub2api-backend:8080/health

# 重新加载配置
docker exec sub2api-caddy caddy reload --config /etc/caddy/Caddyfile
```

---

## 预防措施

### 1. 配置监控告警

推荐使用以下任一方案：
- **Uptime Kuma**（轻量级，Docker 一键部署）
- **UptimeRobot**（免费 SaaS）
- **Prometheus + Grafana + Alertmanager**（完整方案）

监控指标：
- `/health` 端点可用性（每 30 秒检查）
- 容器健康状态
- 系统资源（内存/CPU/磁盘）

### 2. 配置日志聚合

```bash
# 使用 Loki + Promtail 收集日志（可选）
# 或者简单方案：定期备份日志到对象存储
docker logs sub2api-backend > /backup/logs/backend-$(date +%Y%m%d).log
```

### 3. 定期检查资源

```bash
# 添加到 cron，每小时检查一次
0 * * * * docker stats --no-stream --format "{{.Name}}\t{{.MemPerc}}" | \
  awk '$2+0 > 80 {print "WARNING: "$1" 内存超过 80%"}' | \
  mail -s "Docker 资源警告" admin@example.com
```

---

## 联系支持

如果以上方案都无法解决，收集以下信息后寻求支持：
1. `troubleshoot-503.sh` 的完整输出
2. 最近的发布日志：`cat deploy/production/deploy.log | tail -100`
3. 完整的后端日志：`docker logs sub2api-backend --tail 500`
4. 系统信息：`uname -a && free -h && df -h`
