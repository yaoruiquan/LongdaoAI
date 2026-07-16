#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 生产恢复脚本 (OPS-001 恢复演练)
# =============================================================================
# 目标：从备份文件恢复数据库。默认恢复到测试库（sub2api_restore_test），
#       避免误覆盖生产。恢复到生产库需二次确认，属高危操作。
#
# 用法：
#   ./deploy/production/restore.sh <备份文件.sql.gz> [目标库名]
#   ./deploy/production/restore.sh backups/pg_prod_v..._....sql.gz
#   ./deploy/production/restore.sh backups/pg_...sql.gz sub2api      # 恢复到生产库
#   FORCE=1 ./deploy/production/restore.sh backups/pg_...sql.gz      # 跳过交互确认
#
# 退出码：0 成功；非0 失败或用户取消。
# =============================================================================

set -euo pipefail

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"

log()  { printf '\033[1;34m[restore]\033[0m %s\n' "$*"; }
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

# ---- 参数与前置检查 --------------------------------------------------------
BACKUP_FILE="${1:-}"
TARGET_DB="${2:-sub2api_restore_test}"

if [ -z "${BACKUP_FILE}" ]; then
    err "用法：$0 <备份文件.sql.gz> [目标库名，默认 sub2api_restore_test]"
    exit 1
fi
if [ ! -f "${BACKUP_FILE}" ]; then
    err "备份文件不存在：${BACKUP_FILE}"
    exit 1
fi
if [ ! -f "${COMPOSE_FILE}" ]; then err "找不到 compose 文件：${COMPOSE_FILE}"; exit 1; fi
if [ ! -f "${ENV_FILE}" ];    then err "找不到环境文件：${ENV_FILE}"; exit 1; fi

POSTGRES_USER="$(env_val POSTGRES_USER sub2api)"
PROD_DB="$(env_val POSTGRES_DB sub2api)"

IS_PROD_TARGET="0"
if [ "${TARGET_DB}" = "${PROD_DB}" ]; then
    IS_PROD_TARGET="1"
fi

# ---- 危险操作确认 ----------------------------------------------------------
warn "=============================================================="
warn " 恢复操作（高危！会覆盖目标库全部数据）"
warn "   备份文件 : ${BACKUP_FILE}"
warn "   目标库   : ${TARGET_DB}"
warn "   用户     : ${POSTGRES_USER}"
if [ "${IS_PROD_TARGET}" = "1" ]; then
    warn "   *** 目标是生产库（${PROD_DB}）！这将覆盖线上数据！ ***"
fi
warn "=============================================================="

confirm() {
    local prompt="$1" expect="$2" answer=""
    if [ "${FORCE:-0}" = "1" ]; then
        warn "FORCE=1，跳过确认：${prompt}"
        return 0
    fi
    if [ ! -t 0 ]; then
        err "非交互终端且未设置 FORCE=1，拒绝执行危险操作。"
        exit 1
    fi
    printf '%s ' "${prompt}"
    read -r answer
    if [ "${answer}" != "${expect}" ]; then
        err "确认不匹配（需输入 ${expect}），已取消。"
        exit 1
    fi
}

confirm "确认恢复到「${TARGET_DB}」？输入 yes 继续：" "yes"
if [ "${IS_PROD_TARGET}" = "1" ]; then
    confirm "再次确认覆盖生产库「${PROD_DB}」！输入库名以确认：" "${PROD_DB}"
fi

# ---- 执行恢复 --------------------------------------------------------------
log "准备目标库：${TARGET_DB}"
# 若目标非生产测试库，先重建以保证干净恢复；生产库不自动 DROP，需人工评估。
if [ "${IS_PROD_TARGET}" != "1" ]; then
    compose exec -T postgres psql -U "${POSTGRES_USER}" -d postgres \
        -c "DROP DATABASE IF EXISTS \"${TARGET_DB}\";"
    compose exec -T postgres psql -U "${POSTGRES_USER}" -d postgres \
        -c "CREATE DATABASE \"${TARGET_DB}\";"
else
    warn "目标为生产库，跳过自动 DROP/CREATE；将直接导入到现有库。"
fi

log "导入备份到 ${TARGET_DB} ..."
if gunzip -c "${BACKUP_FILE}" | compose exec -T postgres psql -U "${POSTGRES_USER}" -d "${TARGET_DB}"; then
    log "=============================================================="
    log " 恢复完成：${TARGET_DB}"
    log "=============================================================="
    log " 验证提示（人工核对行数是否合理）："
    log "   docker compose -f ${COMPOSE_FILE} --env-file ${ENV_FILE} \\"
    log "     exec -T postgres psql -U ${POSTGRES_USER} -d ${TARGET_DB} \\"
    log "     -c 'SELECT count(*) AS users FROM users; SELECT count(*) AS api_keys FROM api_keys;'"
    exit 0
else
    rc=$?
    err "恢复失败（退出码 ${rc}）。"
    exit "${rc}"
fi
