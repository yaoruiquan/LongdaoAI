#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 生产数据库迁移脚本 (MIG)
# =============================================================================
# 目标：对生产 stack 执行独立的数据库迁移（sub2api -migrate）。
#       迁移与应用启动解耦：先迁移成功，再放量新版本（deploy.sh 依赖此约定）。
#
# 用法：
#   ./deploy/production/migrate.sh
#   COMPOSE_FILE=deploy/production/docker-compose.yml ./deploy/production/migrate.sh
#
# 退出码：
#   0  迁移成功
#   非0 迁移失败（deploy.sh 据此中止发布，不启动新版本）
# =============================================================================

set -euo pipefail

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"

log()  { printf '\033[1;34m[migrate]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

# ---- 前置检查 --------------------------------------------------------------
if [ ! -f "${COMPOSE_FILE}" ]; then
    err "找不到 compose 文件：${COMPOSE_FILE}"
    exit 1
fi
if [ ! -f "${ENV_FILE}" ]; then
    err "找不到环境文件：${ENV_FILE}（生产真实值，不进仓库）"
    exit 1
fi

# 读取镜像版本用于日志（IMAGE_TAG 由 .env 指定）
IMAGE_TAG="$(grep -E '^IMAGE_TAG=' "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- | tr -d '"'"'"' ' || true)"
IMAGE_TAG="${IMAGE_TAG:-<未知>}"

compose() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" "$@"
}

# ---- 执行迁移 --------------------------------------------------------------
START_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
log "=============================================================="
log " 开始数据库迁移"
log "   compose  : ${COMPOSE_FILE}"
log "   env      : ${ENV_FILE}"
log "   版本     : ${IMAGE_TAG}"
log "   开始时间 : ${START_TS}"
log "=============================================================="

# 用一次性容器执行独立迁移命令；--rm 保证不残留容器。
# sub2api -migrate 失败会返回非零退出码，set -e 会让本脚本随之失败。
if compose run --rm sub2api /app/sub2api -migrate; then
    END_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    log "=============================================================="
    log " 迁移成功  版本=${IMAGE_TAG}  结束时间=${END_TS}"
    log "=============================================================="
    exit 0
else
    rc=$?
    err "=============================================================="
    err " 迁移失败（退出码 ${rc}），版本=${IMAGE_TAG}"
    err " 请勿启动新版本应用。排查后重跑本脚本。"
    err "=============================================================="
    exit "${rc}"
fi
