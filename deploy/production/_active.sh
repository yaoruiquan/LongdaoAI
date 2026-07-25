#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 蓝绿部署 · 共享辅助（被其他脚本 source，不单独执行）
# =============================================================================
# 职责：
#   1. 维护「当前活跃色」状态文件 .active_color（值为 blue 或 green）。
#   2. 提供 active_color / inactive_color / svc / compose 等公共函数，
#      让 deploy/migrate/backup/smoke-test/rollback/switch 各脚本都以
#      「活跃色对应的 service 名」来操作，而不再硬编码 sub2api。
#
# 约定：
#   - service 名为 sub2api-blue / sub2api-green。
#   - 首次（状态文件不存在）默认活跃色为 blue。
#   - compose profile 与颜色同名（blue / green），平时只拉起活跃色一个实例。
#
# 用法（在其他脚本顶部）：
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "${SCRIPT_DIR}/_active.sh"
# =============================================================================

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"
# 状态文件放在 compose 同目录，随仓库工作区，但本身不进 git（见 .gitignore）。
ACTIVE_COLOR_FILE="${ACTIVE_COLOR_FILE:-$(dirname "${COMPOSE_FILE}")/.active_color}"

# 读取当前活跃色；缺省 blue。只接受 blue/green，非法值回退 blue 并告警到 stderr。
active_color() {
    local c=""
    if [ -f "${ACTIVE_COLOR_FILE}" ]; then
        c="$(tr -d '[:space:]' < "${ACTIVE_COLOR_FILE}" 2>/dev/null || true)"
    fi
    case "${c}" in
        blue|green) printf '%s' "${c}" ;;
        "")         printf 'blue' ;;
        *)          printf 'blue'; printf '[_active] 非法活跃色 %q，回退 blue\n' "${c}" >&2 ;;
    esac
}

# 与活跃色相对的另一色（部署目标色）。
inactive_color() {
    if [ "$(active_color)" = "blue" ]; then printf 'green'; else printf 'blue'; fi
}

# 把活跃色写入状态文件（切换成功后调用）。
set_active_color() {
    local c="$1"
    case "${c}" in
        blue|green) ;;
        *) printf '[_active] 拒绝写入非法活跃色 %q\n' "${c}" >&2; return 1 ;;
    esac
    printf '%s\n' "${c}" > "${ACTIVE_COLOR_FILE}"
}

# 颜色 -> service 名。
svc() { printf 'sub2api-%s' "$1"; }

# 活跃色 / 目标色对应的 service 名捷径。
active_svc()   { svc "$(active_color)"; }
inactive_svc() { svc "$(inactive_color)"; }

# compose 包装：始终带上 blue+green 两个 profile，使两色 service 都「可寻址」。
# 这样各脚本用具体 service 名操作即可，无需关心 profile：
#   compose exec -T "$(active_svc)" ...      # 对活跃色执行
#   compose up -d "$(inactive_svc)"          # 只拉起目标色（显式 service 名）
#   compose up -d postgres redis caddy       # 拉起共享服务
# 注意：不要用不带 service 名的 `compose up -d`——那会同时拉起 blue+green。
#       蓝绿流程一律显式指定 service 名。
compose() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
        --profile blue --profile green "$@"
}

# 从 .env 读取一个键的值（去引号），$2 为默认值。
env_val() {
    local key="$1" default="${2:-}"
    local v
    v="$(grep -E "^${key}=" "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- | tr -d '"'"'"' ' || true)"
    printf '%s' "${v:-$default}"
}
