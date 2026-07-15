#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 发布后冒烟测试 (TST-001)
# =============================================================================
# 目标：发布后快速验证核心可用性。实现可自动化的 P0/P1 项；
#       需要登录态/管理员/API Key/真实模型调用的项做成可选或占位（不硬编码凭据）。
#
# 用法：
#   ./deploy/production/smoke-test.sh
#   BASE_URL=https://api.example.com ./deploy/production/smoke-test.sh   # 经 caddy 外部地址
#   # 未提供 BASE_URL 时，默认在 sub2api 容器内直连 http://localhost:8080
#
# 退出码：0 全部 P0 通过；非0 任一 P0 失败（deploy.sh 据此终止放量）。
# =============================================================================

set -euo pipefail

# ---- 公共约定（可用环境变量覆盖）------------------------------------------
COMPOSE_FILE="${COMPOSE_FILE:-deploy/production/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-deploy/production/.env}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
# BASE_URL 为空 => 容器内直连；非空 => 用 curl 打外部地址（经 caddy）
BASE_URL="${BASE_URL:-}"

log()  { printf '\033[1;34m[smoke]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }
todo() { printf '\033[1;36m[TODO]\033[0m %s\n' "$*"; }

compose() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" "$@"
}

env_val() {
    local key="$1" default="${2:-}"
    local v
    v="$(grep -E "^${key}=" "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- | tr -d '"'"'"' ' || true)"
    printf '%s' "${v:-$default}"
}

IMAGE_TAG="$(env_val IMAGE_TAG unknown 2>/dev/null || echo unknown)"
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ---- HTTP 请求封装 ---------------------------------------------------------
# 返回：把 HTTP 状态码打印到 stdout（末行），响应体打印到 stderr（调试）。
# 有 BASE_URL 时用宿主机 curl；否则在 sub2api 容器内用 wget 访问 localhost。
http_code() {
    local path="$1" code
    if [ -n "${BASE_URL}" ]; then
        code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "${BASE_URL}${path}" || echo 000)"
    else
        # 容器内 wget（busybox）：--server-response 输出到 stderr，抓 HTTP/ 状态码
        code="$(compose exec -T sub2api sh -c \
            "wget -q -T 10 -O /dev/null -S 'http://localhost:8080${path}' 2>&1 | awk '/HTTP\\//{c=\$2} END{print c}'" \
            2>/dev/null || echo 000)"
        code="${code:-000}"
    fi
    printf '%s' "${code}"
}

http_body() {
    local path="$1"
    if [ -n "${BASE_URL}" ]; then
        curl -s --max-time 10 "${BASE_URL}${path}" || true
    else
        compose exec -T sub2api sh -c "wget -q -T 10 -O - 'http://localhost:8080${path}'" 2>/dev/null || true
    fi
}

FAILED=0
fail_p0() { err "P0 失败：$*"; FAILED=1; }

log "=============================================================="
log " 冒烟测试开始"
log "   环境   : ${ENVIRONMENT}"
log "   版本   : ${IMAGE_TAG}"
log "   时间   : ${NOW}"
log "   目标   : ${BASE_URL:-容器内 http://localhost:8080}"
log "=============================================================="

# ---- P0-1: /health 返回 200（必测）----------------------------------------
log "[P0] 检查 /health ..."
CODE="$(http_code /health)"
if [ "${CODE}" = "200" ]; then
    BODY="$(http_body /health)"
    if printf '%s' "${BODY}" | grep -q '"status"'; then
        log "  /health OK (200, body 含 status)"
    else
        log "  /health OK (200)"
    fi
else
    fail_p0 "/health 返回 ${CODE}（期望 200）"
fi

# ---- P1-1: 首页/静态资源可访问（HTTP 200）--------------------------------
log "[P1] 检查首页 / ..."
CODE="$(http_code /)"
if [ "${CODE}" = "200" ] || [ "${CODE}" = "301" ] || [ "${CODE}" = "302" ]; then
    log "  首页可访问 (${CODE})"
else
    warn "  首页返回 ${CODE}（期望 200/3xx），标记为 P1 警告，不阻断发布。"
fi

# ---- P1-2: 注册关闭策略校验 -----------------------------------------------
# 后端开启「后台模式」时，BackendModeAuthGuard 会拦截 /api/v1/auth/register，
# 期望返回非 200（403/404）。若你的环境开放注册，则应返回可注册相关状态。
log "[P1] 检查注册接口策略 /api/v1/auth/register ..."
REG_CODE="$(http_code /api/v1/auth/register)"
# 仅 GET 探测：接口是 POST，GET 通常 404/405；此处主要验证「未被开放为可匿名注册页」。
if [ "${REG_CODE}" = "403" ] || [ "${REG_CODE}" = "404" ] || [ "${REG_CODE}" = "405" ]; then
    log "  注册入口按预期受限/非开放 (${REG_CODE})"
else
    warn "  注册入口返回 ${REG_CODE}，请人工确认注册开关策略是否符合预期。"
fi

# ---- 需要凭据的项：占位 TODO（自动化受限，不硬编码任何凭据）---------------
todo "以下项需要凭据/登录态，未自动化，请按发布计划人工执行或补全脚本："
todo "  - 管理员登录 + TOTP 二次验证（需 ADMIN 账号与 TOTP，勿写入脚本）"
todo "  - 创建测试用户 / 生成 API Key（需登录态）"
todo "  - 用 API Key 发起一次真实模型调用（如 /v1/chat/completions），校验计费与限流"
todo "  - 订阅/额度扣减、风控事件写入等业务链路核对"
todo "  可通过在 CI/受控环境中注入临时凭据（环境变量/密钥管理）后扩展本脚本。"

# ---- 结果汇总 --------------------------------------------------------------
END="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [ "${FAILED}" -eq 0 ]; then
    log "=============================================================="
    log " 冒烟测试通过  环境=${ENVIRONMENT}  版本=${IMAGE_TAG}  时间=${END}"
    log "=============================================================="
    exit 0
else
    err "=============================================================="
    err " 冒烟测试存在 P0 失败  环境=${ENVIRONMENT}  版本=${IMAGE_TAG}  时间=${END}"
    err " 建议立即回滚（rollback.sh）。"
    err "=============================================================="
    exit 1
fi
