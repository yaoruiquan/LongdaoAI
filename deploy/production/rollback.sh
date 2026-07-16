#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 回滚脚本 (OPS-002)
# =============================================================================
# 目标：把应用镜像切回上一个不可变版本 tag 并重启，快速恢复服务。
#
# 关键安全约束（spec 硬要求）：
#   * 绝不删除 postgres / redis 数据卷。本脚本只更新 sub2api 应用镜像，
#     不执行 down -v、不 docker volume rm，不触碰任何数据卷。
#
# 迁移说明（OPS-002）：
#   * 迁移默认「前向修复」：回滚仅切换应用镜像，不自动回退数据库结构。
#   * 若上一次发布包含「不兼容迁移」（删列/改类型等），仅切镜像可能无法工作，
#     此时需先用 restore.sh 从发布前备份恢复数据库，再切镜像。请谨慎评估。
#
# 用法：
#   ./deploy/production/rollback.sh <上一个IMAGE_TAG>
#   ./deploy/production/rollback.sh 2026.07.15-1
#
# 实现方式：通过 IMAGE_TAG=<旧版本> 覆盖环境变量执行 up -d，
#           不修改 .env 文件本身（保持 .env 干净、可追溯）。
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"
IMAGE_REPO="${IMAGE_REPO:-longdao/sub2api}"
DEPLOY_LOG="${DEPLOY_LOG:-deploy/production/deploy.log}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-120}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-5}"

log()  { printf '\033[1;34m[rollback]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

record() { printf '%s\n' "$1" >> "${DEPLOY_LOG}"; }

# ---- 参数与前置检查 --------------------------------------------------------
ROLLBACK_TAG="${1:-}"
if [ -z "${ROLLBACK_TAG}" ]; then
    err "用法：$0 <要回滚到的镜像版本 tag，例如 2026.07.15-1>"
    exit 1
fi
[ -f "${COMPOSE_FILE}" ] || { err "找不到 compose 文件：${COMPOSE_FILE}"; exit 1; }
[ -f "${ENV_FILE}" ]     || { err "找不到环境文件：${ENV_FILE}"; exit 1; }

TARGET_IMAGE="${IMAGE_REPO}:${ROLLBACK_TAG}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

# 用 IMAGE_TAG 覆盖变量执行 compose（不改 .env）
compose() {
    IMAGE_TAG="${ROLLBACK_TAG}" docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" "$@"
}

log "=============================================================="
log " 龙道 AI 回滚开始"
log "   目标镜像 : ${TARGET_IMAGE}"
log "   数据卷   : 不会被删除或重建（仅切换应用镜像）"
log "=============================================================="

# 检查目标镜像存在
if ! docker image inspect "${TARGET_IMAGE}" >/dev/null 2>&1; then
    err "本地找不到回滚目标镜像 ${TARGET_IMAGE}。"
    err "请先 docker pull ${TARGET_IMAGE} 或确认该历史版本仍在本机。"
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) ROLLBACK FAILED tag=${ROLLBACK_TAG} reason=\"image not found\""
    exit 1
fi

mkdir -p "$(dirname "${DEPLOY_LOG}")"
record "$(date -u +%Y-%m-%dT%H:%M:%SZ) ROLLBACK START tag=${ROLLBACK_TAG} commit=${COMMIT}"

# ---- 切换镜像并重启（仅 sub2api，不动数据卷）------------------------------
log "以 IMAGE_TAG=${ROLLBACK_TAG} 重新拉起 sub2api ..."
if ! compose up -d sub2api; then
    err "compose up -d 失败。"
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) ROLLBACK FAILED tag=${ROLLBACK_TAG} reason=\"compose up failed\""
    exit 1
fi

# ---- 健康检查 --------------------------------------------------------------
log "健康检查（超时 ${HEALTH_TIMEOUT}s）..."
deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
healthy=0
while [ "$(date +%s)" -lt "${deadline}" ]; do
    status="$(compose ps --format '{{.Health}}' sub2api 2>/dev/null | head -n1 || true)"
    if [ "${status}" = "healthy" ]; then healthy=1; break; fi
    if compose exec -T sub2api wget -q -T 5 -O /dev/null "http://localhost:8080/health" >/dev/null 2>&1; then
        healthy=1; break
    fi
    sleep "${HEALTH_INTERVAL}"
done

END_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [ "${healthy}" -ne 1 ]; then
    err "回滚后健康检查失败，最近日志："
    compose logs --tail=50 sub2api >&2 || true
    record "${END_TS} ROLLBACK FAILED tag=${ROLLBACK_TAG} commit=${COMMIT} reason=\"health check failed\""
    err "回滚后服务仍不健康，请人工介入（可能涉及不兼容迁移，需先 restore.sh 恢复数据库）。"
    exit 1
fi

record "${END_TS} ROLLBACK SUCCESS tag=${ROLLBACK_TAG} commit=${COMMIT} result=ok"
log "=============================================================="
log " 回滚成功  已切回版本=${ROLLBACK_TAG}  时间=${END_TS}"
log "   提示：如需持久化该版本，请把 .env 的 IMAGE_TAG 也改为 ${ROLLBACK_TAG}。"
log "   已记录到 ${DEPLOY_LOG}"
log "=============================================================="
exit 0
