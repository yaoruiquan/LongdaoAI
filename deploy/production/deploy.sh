#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 标准发布流程编排（蓝绿零中断版，呼应 spec §10.1）
# =============================================================================
# 编排顺序（任一步失败即停止并提示回滚）：
#   1. 检查 .env 存在、IMAGE_TAG 已设置
#   2. 检查目标镜像存在（docker image inspect）
#   3. 发布前备份（backup.sh）
#   4. 执行迁移（migrate.sh，对目标色一次性容器）；失败则中止，不放量
#   5. 蓝绿切换（switch.sh）：拉起目标色 → 健康检查 → 冒烟 → 放量 → 停旧色
#   6. 记录发布结果到 deploy.log
#
# 与旧版差异：不再 docker compose up -d 直接重建单实例（那会产生 502 停机窗口）；
# 改为蓝绿切换，全程至少一个健康实例在线，实现零中断发布。
#
# 用法：
#   ./deploy/production/deploy.sh
#   SKIP_BACKUP=1 ./deploy/production/deploy.sh     # 跳过发布前备份（不推荐）
#   KEEP_OLD=1 ./deploy/production/deploy.sh         # 切完保留旧色（便于快速回退）
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_active.sh
source "${SCRIPT_DIR}/_active.sh"   # 提供 compose()/env_val()/active_svc()/inactive_svc() 等

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
IMAGE_REPO="${IMAGE_REPO:-longdao/sub2api}"
DEPLOY_LOG="${DEPLOY_LOG:-deploy/production/deploy.log}"

log()  { printf '\033[1;34m[deploy]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

record() {
    # 记录一行发布结果到 deploy.log
    printf '%s\n' "$1" >> "${DEPLOY_LOG}"
}

abort() {
    err "$*"
    err ">>> 发布已中止。如已启动新色，请运行：deploy/production/rollback.sh <上一版本tag>"
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
# ---- 5. 蓝绿切换（拉起目标色 → 健康 → 冒烟 → 放量 → 停旧色）-----------
log "[5/6] 蓝绿切换（switch.sh）..."
log "      当前活跃色: $(active_color)  →  目标色: $(inactive_color)"
if ! COMPOSE_FILE="${COMPOSE_FILE}" ENV_FILE="${ENV_FILE}" \
        DEPLOY_LOG="${DEPLOY_LOG}" \
        bash "${SCRIPT_DIR}/switch.sh"; then
    abort "蓝绿切换失败（健康检查/冒烟/放量任一步未通过）。"
fi

# ---- 6. 记录发布结果 -------------------------------------------------------
END_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
record "${END_TS} DEPLOY SUCCESS tag=${IMAGE_TAG} commit=${COMMIT} result=ok"
log "=============================================================="
log " 发布成功  版本=${IMAGE_TAG}  commit=${COMMIT}  时间=${END_TS}"
log "   已记录到 ${DEPLOY_LOG}"
log "=============================================================="
exit 0
