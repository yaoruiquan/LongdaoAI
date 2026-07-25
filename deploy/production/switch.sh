#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 蓝绿切换编排 (零中断发布核心)
# =============================================================================
# 流程：
#   1. 定位当前活跃色 CUR 与目标色 TARGET（= 另一色）。
#   2. 用 .env 中的（新）IMAGE_TAG 拉起 TARGET 实例。
#   3. 轮询 TARGET 变 healthy。
#   4. 容器内直连对 TARGET 冒烟（放量前验证）；失败即停 TARGET 回滚、不切流量。
#   5. 观察窗口：Caddy 通过主动健康检查把流量导向 TARGET
#      （green→blue 在 TARGET healthy 即切；blue→green 在停 CUR 后切，见下）。
#   6. 停掉 CUR 实例（SIGTERM → 应用 60s 优雅 drain 在途流式请求）。
#   7. 写入新活跃色到 .active_color。
#
# 关于 lb_policy first 的方向不对称（已在流程中用「先冒烟再放量」消化）：
#   Caddy 按 Caddyfile 顺序 (blue 先 green 后) 选第一个健康 upstream。
#     - green->blue：blue healthy 瞬间即被选中放量（green 尚在跑，零丢请求）。
#     - blue->green：blue 仍健康时 first 继续走 blue，直到停 blue 才切 green。
#   两方向全程至少一个健康实例在线，故无中断。第 4 步的容器内冒烟保证
#   「放量的 TARGET 一定已通过健康检查与冒烟」。
#
# 用法：
#   ./deploy/production/switch.sh                 # 切到另一色（用 .env 当前 IMAGE_TAG）
#   OBSERVE_SECONDS=15 ./deploy/production/switch.sh
#   KEEP_OLD=1 ./deploy/production/switch.sh       # 切完不停旧色（保留以便快速回退）
#
# 退出码：0 成功；非0 失败（失败时保证 CUR 仍在线，不影响现网）。
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_active.sh
source "${SCRIPT_DIR}/_active.sh"

HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-120}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-3}"
OBSERVE_SECONDS="${OBSERVE_SECONDS:-10}"
KEEP_OLD="${KEEP_OLD:-0}"
DEPLOY_LOG="${DEPLOY_LOG:-deploy/production/deploy.log}"

log()  { printf '\033[1;34m[switch]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }
record() { mkdir -p "$(dirname "${DEPLOY_LOG}")"; printf '%s\n' "$1" >> "${DEPLOY_LOG}"; }

# 等待某个 service 变 healthy（compose 报告的 health，兜底容器内直连 /health）。
wait_healthy() {
    local svc="$1" deadline
    deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
    while [ "$(date +%s)" -lt "${deadline}" ]; do
        local status
        status="$(compose ps --format '{{.Health}}' "${svc}" 2>/dev/null | head -n1 || true)"
        if [ "${status}" = "healthy" ]; then return 0; fi
        if compose exec -T "${svc}" wget -q -T 5 -O /dev/null "http://localhost:8080/health" >/dev/null 2>&1; then
            return 0
        fi
        sleep "${HEALTH_INTERVAL}"
    done
    return 1
}

# 判断某 service 当前是否有运行中的容器。
is_running() {
    local svc="$1" id
    id="$(compose ps -q "${svc}" 2>/dev/null | head -n1 || true)"
    [ -n "${id}" ]
}

# ---- 前置检查 --------------------------------------------------------------
[ -f "${COMPOSE_FILE}" ] || { err "找不到 compose 文件：${COMPOSE_FILE}"; exit 1; }
[ -f "${ENV_FILE}" ]     || { err "找不到环境文件：${ENV_FILE}"; exit 1; }

# IMAGE_TAG 优先取环境变量覆盖（rollback.sh 用旧 tag 调本脚本实现零中断回滚），
# 否则回退 .env。docker compose 变量替换中 shell 环境变量也优先于 --env-file，
# 故导出 IMAGE_TAG 后 compose up 会用该值拉起目标色。
IMAGE_TAG="${IMAGE_TAG:-$(env_val IMAGE_TAG)}"
export IMAGE_TAG
[ -n "${IMAGE_TAG}" ] || { err "未设置 IMAGE_TAG（.env 或环境变量）"; exit 1; }

CUR_COLOR="$(active_color)"
TARGET_COLOR="$(inactive_color)"
CUR_SVC="$(svc "${CUR_COLOR}")"
TARGET_SVC="$(svc "${TARGET_COLOR}")"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

log "=============================================================="
log " 蓝绿切换开始"
log "   当前活跃色 : ${CUR_COLOR} (${CUR_SVC})"
log "   目标色     : ${TARGET_COLOR} (${TARGET_SVC})"
log "   目标镜像   : ${IMAGE_REPO:-longdao/sub2api}:${IMAGE_TAG}"
log "   commit     : ${COMMIT}"
log "=============================================================="
record "$(date -u +%Y-%m-%dT%H:%M:%SZ) SWITCH START ${CUR_COLOR}->${TARGET_COLOR} tag=${IMAGE_TAG} commit=${COMMIT}"

# ---- 1. 确保共享服务在线（postgres/redis/caddy）---------------------------
log "[1/6] 确保共享服务在线（postgres / redis / caddy）..."
compose up -d postgres redis caddy

# ---- 2. 拉起目标色（用新 IMAGE_TAG）---------------------------------------
log "[2/6] 拉起目标色实例 ${TARGET_SVC}（IMAGE_TAG=${IMAGE_TAG}）..."
if ! compose up -d "${TARGET_SVC}"; then
    err "拉起 ${TARGET_SVC} 失败。现网仍由 ${CUR_SVC} 提供服务，未受影响。"
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) SWITCH FAILED reason=\"up target failed\""
    exit 1
fi

# ---- 3. 等目标色 healthy ----------------------------------------------------
log "[3/6] 等待 ${TARGET_SVC} 健康（超时 ${HEALTH_TIMEOUT}s）..."
if ! wait_healthy "${TARGET_SVC}"; then
    err "${TARGET_SVC} 未在 ${HEALTH_TIMEOUT}s 内变健康，最近日志："
    compose logs --tail=50 "${TARGET_SVC}" >&2 || true
    warn "回滚：停掉未就绪的 ${TARGET_SVC}，现网继续由 ${CUR_SVC} 提供服务。"
    compose stop "${TARGET_SVC}" >/dev/null 2>&1 || true
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) SWITCH FAILED reason=\"target unhealthy\""
    exit 1
fi
log "  ${TARGET_SVC} 健康。"

# ---- 4. 放量前冒烟（容器内直连目标色）-------------------------------------
log "[4/6] 对 ${TARGET_SVC} 冒烟（容器内直连，放量前验证）..."
if ! SMOKE_SVC="${TARGET_SVC}" BASE_URL="" \
        COMPOSE_FILE="${COMPOSE_FILE}" ENV_FILE="${ENV_FILE}" \
        bash "${SCRIPT_DIR}/smoke-test.sh"; then
    err "${TARGET_SVC} 冒烟未通过。回滚：停掉目标色，现网继续由 ${CUR_SVC} 提供服务。"
    compose stop "${TARGET_SVC}" >/dev/null 2>&1 || true
    record "$(date -u +%Y-%m-%dT%H:%M:%SZ) SWITCH FAILED reason=\"target smoke failed\""
    exit 1
fi
log "  ${TARGET_SVC} 冒烟通过。"

# ---- 5. 观察窗口：让 Caddy 主动健康检查把流量导向目标色 -------------------
# green->blue：blue healthy 时 first 已放量到 blue（CUR=green 仍在线，零丢）。
# blue->green：first 仍走 blue，直到第 6 步停 blue 才切 green。
log "[5/6] 观察 ${OBSERVE_SECONDS}s，确认 ${TARGET_SVC} 承接正常..."
sleep "${OBSERVE_SECONDS}"

# ---- 6. 停掉旧色（优雅 drain）+ 写活跃色 -----------------------------------
if [ "${KEEP_OLD}" = "1" ]; then
    warn "[6/6] KEEP_OLD=1：保留旧色 ${CUR_SVC} 运行（便于快速回退）。"
    warn "      注意：blue->green 方向下，只要 blue 仍健康，first 会继续走 blue，"
    warn "      流量不会切到 green！确认无误后请手动：compose stop ${CUR_SVC}"
else
    if is_running "${CUR_SVC}"; then
        log "[6/6] 停掉旧色 ${CUR_SVC}（SIGTERM → 应用 60s 优雅 drain 在途请求）..."
        # -t 95 > 应用 60s 关闭超时 + 余量，确保 Docker 不提前 SIGKILL。
        compose stop -t 95 "${CUR_SVC}" || warn "停 ${CUR_SVC} 返回非零，请人工确认。"
    else
        log "[6/6] 旧色 ${CUR_SVC} 未在运行（首次部署/引导场景），跳过停旧。"
    fi
fi

set_active_color "${TARGET_COLOR}"
END_TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
record "${END_TS} SWITCH SUCCESS ${CUR_COLOR}->${TARGET_COLOR} tag=${IMAGE_TAG} commit=${COMMIT} result=ok"
log "=============================================================="
log " 切换成功  活跃色 ${CUR_COLOR} -> ${TARGET_COLOR}  版本=${IMAGE_TAG}  时间=${END_TS}"
[ "${KEEP_OLD}" = "1" ] && log "   旧色 ${CUR_SVC} 仍在运行（KEEP_OLD=1）。"
log "=============================================================="
exit 0
