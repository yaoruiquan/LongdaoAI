#!/usr/bin/env bash
# =============================================================================
# 503 故障快速排查脚本
# =============================================================================
# 用法：在生产服务器上运行
#   ssh root@longdaoai.cn 'bash -s' < deploy/production/troubleshoot-503.sh
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[检查]${NC} $*"; }
warn() { echo -e "${YELLOW}[警告]${NC} $*"; }
err() { echo -e "${RED}[错误]${NC} $*"; }

echo "=============================================================="
echo " 503 故障排查开始 - $(date)"
echo "=============================================================="

# ---- 1. 检查容器状态 ----------------------------------------------------
log "[1] 检查容器运行状态..."
docker compose -f deploy/production/docker-compose.yml ps

# ---- 2. 检查健康检查状态 ------------------------------------------------
log "[2] 检查后端容器健康状态..."
BACKEND_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' sub2api-backend 2>/dev/null || echo "not_found")
echo "后端健康状态: ${BACKEND_HEALTH}"

if [ "${BACKEND_HEALTH}" != "healthy" ]; then
    warn "后端容器不健康！最近 5 次健康检查日志："
    docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' sub2api-backend | tail -20
fi

# ---- 3. 检查后端日志（最近 100 行）--------------------------------------
log "[3] 后端容器日志（最近 100 行）..."
docker logs sub2api-backend --tail 100 2>&1 | tail -50

# ---- 4. 检查 Caddy 日志（反向代理错误）---------------------------------
log "[4] Caddy 日志（最近 50 行）..."
docker logs sub2api-caddy --tail 50 2>&1 | grep -E "503|error|upstream" || echo "无 503 相关日志"

# ---- 5. 检查数据库连接 --------------------------------------------------
log "[5] 检查数据库连接..."
docker exec sub2api-postgres pg_isready -U sub2api || warn "PostgreSQL 不可达"

# ---- 6. 检查 Redis 连接 -------------------------------------------------
log "[6] 检查 Redis 连接..."
docker exec sub2api-redis redis-cli ping || warn "Redis 不可达"

# ---- 7. 检查系统资源 ----------------------------------------------------
log "[7] 系统资源使用情况..."
echo "内存使用："
free -h
echo ""
echo "磁盘使用："
df -h | grep -E "/$|/var/lib/docker"
echo ""
echo "Docker 容器资源："
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# ---- 8. 检查网络连通性 --------------------------------------------------
log "[8] 检查内部网络连通性..."
docker exec sub2api-backend wget -q -O- http://localhost:8080/health || warn "后端 /health 端点不可达"

# ---- 9. 检查最近的重启记录 ----------------------------------------------
log "[9] 容器最近重启记录..."
docker inspect --format='{{.Name}} - 重启次数: {{.RestartCount}}, 最后启动: {{.State.StartedAt}}' \
    $(docker ps -q --filter "name=sub2api")

echo "=============================================================="
echo " 排查完成 - $(date)"
echo "=============================================================="
