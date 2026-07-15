#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 生产备份脚本 (OPS-001)
# =============================================================================
# 目标：备份 PostgreSQL 数据库 + 应用 /app/data 目录，带时间戳/环境/版本命名，
#       支持保留策略（保留最近 N 份）。备份至少一份需复制到服务器之外。
#
# 用法：
#   ./deploy/production/backup.sh
#   KEEP=30 ./deploy/production/backup.sh          # 保留最近 30 份
#   BACKUP_DIR=/mnt/backups ./deploy/production/backup.sh
#
# 退出码：0 全部成功；非0 任一环节失败（供监控告警）。
# =============================================================================

set -euo pipefail

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"
BACKUP_DIR="${BACKUP_DIR:-deploy/production/backups}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
KEEP="${KEEP:-14}"

log()  { printf '\033[1;34m[backup]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

compose() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" "$@"
}

# 从 .env 读取一个键的值（去掉引号），$2 为默认值
env_val() {
    local key="$1" default="${2:-}"
    local v
    v="$(grep -E "^${key}=" "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- | tr -d '"'"'"' ' || true)"
    printf '%s' "${v:-$default}"
}

# ---- 前置检查 --------------------------------------------------------------
if [ ! -f "${COMPOSE_FILE}" ]; then err "找不到 compose 文件：${COMPOSE_FILE}"; exit 1; fi
if [ ! -f "${ENV_FILE}" ];    then err "找不到环境文件：${ENV_FILE}"; exit 1; fi

POSTGRES_USER="$(env_val POSTGRES_USER sub2api)"
POSTGRES_DB="$(env_val POSTGRES_DB sub2api)"
IMAGE_TAG="$(env_val IMAGE_TAG unknown)"

mkdir -p "${BACKUP_DIR}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
# 命名含 环境 + 版本 + 时间戳，例如 pg_prod_v2026.07.15-1_20260715T143000Z.sql.gz
BASE="${ENVIRONMENT}_v${IMAGE_TAG}_${TS}"
PG_FILE="${BACKUP_DIR}/pg_${BASE}.sql.gz"
DATA_FILE="${BACKUP_DIR}/data_${BASE}.tar.gz"

log "=============================================================="
log " 开始备份  环境=${ENVIRONMENT}  版本=${IMAGE_TAG}  时间=${TS}"
log "   数据库   : ${POSTGRES_DB} (user=${POSTGRES_USER})"
log "   备份目录 : ${BACKUP_DIR}"
log "   保留份数 : ${KEEP}"
log "=============================================================="

# ---- 1) PostgreSQL 备份（pg_dump | gzip）----------------------------------
# 用 -T 关闭 TTY 分配，便于非交互管道；密码由 postgres 容器内环境提供。
log "导出数据库 -> ${PG_FILE}"
if ! compose exec -T postgres pg_dump -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
        | gzip -c > "${PG_FILE}"; then
    err "数据库备份失败。"
    rm -f "${PG_FILE}"
    exit 1
fi
# 校验产物非空
if [ ! -s "${PG_FILE}" ]; then
    err "数据库备份产物为空：${PG_FILE}"
    rm -f "${PG_FILE}"
    exit 1
fi
log "数据库备份完成：$(du -h "${PG_FILE}" | cut -f1)"

# ---- 2) /app/data 备份（容器内 tar -> gzip）-------------------------------
# 从 sub2api 容器内把 /app/data 打包输出到宿主机，避免依赖卷挂载路径。
log "打包 /app/data -> ${DATA_FILE}"
if ! compose exec -T sub2api tar -czf - -C /app data > "${DATA_FILE}"; then
    err "/app/data 备份失败。"
    rm -f "${DATA_FILE}"
    exit 1
fi
if [ ! -s "${DATA_FILE}" ]; then
    err "/app/data 备份产物为空：${DATA_FILE}"
    rm -f "${DATA_FILE}"
    exit 1
fi
log "/app/data 备份完成：$(du -h "${DATA_FILE}" | cut -f1)"

# ---- 3) 保留策略：仅保留最近 KEEP 份（按前缀分别清理）--------------------
prune() {
    local prefix="$1"
    # 按修改时间倒序列出，跳过前 KEEP 份，删除其余
    local files
    files="$(ls -1t "${BACKUP_DIR}/${prefix}"_*.gz 2>/dev/null || true)"
    [ -z "${files}" ] && return 0
    printf '%s\n' "${files}" | tail -n +"$((KEEP + 1))" | while IFS= read -r old; do
        [ -n "${old}" ] || continue
        log "清理过期备份：${old}"
        rm -f "${old}"
    done
}
prune "pg"
prune "data"

# ---- 4) 异地复制提示（OPS-001：至少一份复制到服务器之外）------------------
# 生产强烈建议把备份同步到对象存储或另一台机器/磁盘，防止单机损坏丢失。
# 取消注释并按实际环境配置其一：
#
#   # rsync 到另一台服务器/磁盘：
#   # rsync -avz "${PG_FILE}" "${DATA_FILE}" backup-host:/data/longdao-backups/
#
#   # 同步到对象存储（S3 兼容）：
#   # aws s3 cp "${PG_FILE}"   "s3://longdao-backups/${ENVIRONMENT}/"
#   # aws s3 cp "${DATA_FILE}" "s3://longdao-backups/${ENVIRONMENT}/"
#
# 也可通过环境变量提供一个 hook 命令，对每个产物执行异地复制：
if [ -n "${OFFSITE_HOOK:-}" ]; then
    log "执行异地复制 hook：${OFFSITE_HOOK}"
    for f in "${PG_FILE}" "${DATA_FILE}"; do
        if ! "${OFFSITE_HOOK}" "${f}"; then
            err "异地复制失败：${f}（本地备份已保留，请人工处理）"
            exit 1
        fi
    done
else
    warn "未配置异地复制（OFFSITE_HOOK 未设置）。请确保至少一份备份复制到服务器之外。"
fi

log "=============================================================="
log " 备份成功"
log "   数据库   : ${PG_FILE}"
log "   数据目录 : ${DATA_FILE}"
log "=============================================================="
exit 0
