#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 标准发布流程编排 (呼应 spec §10.1)
# =============================================================================
# 编排顺序（任一步失败即停止并提示回滚）：
#   1. 检查 .env 存在、IMAGE_TAG 已设置
#   2. 检查目标镜像存在（docker image inspect）
#   3. 发布前备份（backup.sh）
#   4. 执行迁移（migrate.sh）；失败则中止，不启动新版本
#   5. docker compose up -d
#   6. 健康检查（轮询容器 health / /health）
#   7. 冒烟测试（smoke-test.sh）
#   8. 记录发布结果到 deploy.log
#
# 用法：
#   ./deploy/production/deploy.sh
#   SKIP_BACKUP=1 ./deploy/production/deploy.sh    # 跳过发布前备份（不推荐）
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"
IMAGE_REPO="${IMAGE_REPO:-longdao/sub2api}"
DEPLOY_LOG="${DEPLOY_LOG:-deploy/production/deploy.log}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-120}"   # 健康检查总超时（秒）
HEALTH_INTERVAL="${HEALTH_INTERVAL:-5}"   # 轮询间隔（秒）

log()  { printf '\033[1;34m[deploy]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

compose() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" "$@"
}

env_val() {
    local key="$1" default="${2:-}"
    local v
    v="$(grep -E "^${key}=" "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- | tr -d '"'"'"' ' || true)"
    printf '%s' "${v:-$default}"
}

record() {
    # 记录一行发布结果到 deploy.log
    printf '%s\n' "$1" >> "${DEPLOY_LOG}"
}

abort() {
    err "$*"
    err ">>> 发布已中止。如已启动新版本，请运行：deploy/production/rollback.sh <上一版本tag>"
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) DEPLOY FAILED tag=${IMAGE_TAG:-?} commit=${COMMIT:-?} reason=\"$*\""
    exit 1
}

# ---- 1. 检查 .env / IMAGE_TAG ---------------------------------------------
[ -f "${COMPOSE_FILE}" ] || abort "找不到 compose 文件：${COMPOSE_FILE}"
[ -f "${ENV_FILE}" ]     || abort "找不到环境文件：${ENV_FILE}（生产真实值，需先创建）"

IMAGE_TAG="$(env_val IMAGE_TAG)"
[ -n "${IMAGE_TAG}" ] || abort ".env 中未设置 IMAGE_TAG（发布版本必须显式指定不可变 tag）"

COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
TARGET_IMAGE="${IMAGE_REPO}:${IMAGE_TAG}"

log "=============================================================="
log " 龙道 AI 发布开始"
log "   镜像   : ${TARGET_IMAGE}"
log "   commit : ${COMMIT}"
log "   compose: ${COMPOSE_FILE}"
log "=============================================================="
mkdir -p "$(dirname "${DEPLOY_LOG}")"
record "$(date -u +%Y-%m-%dT%H:%M:%SZ) DEPLOY START tag=${IMAGE_TAG} commit=${COMMIT}"

# ---- 2. 检查目标镜像存在 ---------------------------------------------------
log "[2/8] 检查目标镜像存在：${TARGET_IMAGE}"
if ! docker image inspect "${TARGET_IMAGE}" >/dev/null 2>&1; then
    abort "本地找不到镜像 ${TARGET_IMAGE}。请先构建：deploy/build_production_image.sh ${IMAGE_TAG}（或 docker pull）"
fi

# ---- 3. 发布前备份 ---------------------------------------------------------
if [ "${SKIP_BACKUP:-0}" = "1" ]; then
    warn "[3/8] SKIP_BACKUP=1，跳过发布前备份（不推荐）。"
else
    log "[3/8] 发布前备份 ..."
    if ! COMPOSE_FILE="${COMPOSE_FILE}" ENV_FILE="${ENV_FILE}" bash "${SCRIPT_DIR}/backup.sh"; then
        abort "发布前备份失败，中止发布。"
    fi
fi

# ---- 4. 执行迁移（失败则中止，不启动新版本）------------------------------
log "[4/8] 执行数据库迁移 ..."
if ! COMPOSE_FILE="${COMPOSE_FILE}" ENV_FILE="${ENV_FILE}" bash "${SCRIPT_DIR}/migrate.sh"; then
    abort "数据库迁移失败，按 spec 硬要求不启动新版本。"
fi

# ---- 5. 启动/更新服务 ------------------------------------------------------
log "[5/8] docker compose up -d ..."
if ! compose up -d; then
    abort "docker compose up -d 失败。"
fi

# ---- 6. 健康检查（轮询）---------------------------------------------------
log "[6/8] 健康检查（超时 ${HEALTH_TIMEOUT}s）..."
deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
healthy=0
while [ "$(date +%s)" -lt "${deadline}" ]; do
    # 优先看 compose 报告的容器 health 状态
    status="$(compose ps --format '{{.Health}}' sub2api 2>/dev/null | head -n1 || true)"
    if [ "${status}" = "healthy" ]; then
        healthy=1; break
    fi
    # 兜底：容器内直连 /health
    if compose exec -T sub2api wget -q -T 5 -O /dev/null "http://localhost:8080/health" >/dev/null 2>&1; then
        healthy=1; break
    fi
    sleep "${HEALTH_INTERVAL}"
done
if [ "${healthy}" -ne 1 ]; then
    err "健康检查失败，最近日志："
    compose logs --tail=50 sub2api >&2 || true
    abort "服务在 ${HEALTH_TIMEOUT}s 内未变为 healthy。"
fi
log "  服务健康。"

# ---- 7. 冒烟测试 -----------------------------------------------------------
log "[7/8] 冒烟测试 ..."
if ! COMPOSE_FILE="${COMPOSE_FILE}" ENV_FILE="${ENV_FILE}" bash "${SCRIPT_DIR}/smoke-test.sh"; then
    abort "冒烟测试未通过（P0 失败）。"
fi

# ---- 8. 记录发布结果 -------------------------------------------------------
END_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
record "${END_TS} DEPLOY SUCCESS tag=${IMAGE_TAG} commit=${COMMIT} result=ok"
log "=============================================================="
log " 发布成功  版本=${IMAGE_TAG}  commit=${COMMIT}  时间=${END_TS}"
log "   已记录到 ${DEPLOY_LOG}"
log "=============================================================="
exit 0
