#!/usr/bin/env bash
# =============================================================================
# 龙道 AI 生产镜像构建脚本 (REL-001 / REL-002)
# =============================================================================
# 目标：用包含本地龙道 AI 改造的源码，构建“不可变版本号 + 精确映射到 git commit”
#       的自有镜像，替代上游 weishaw/sub2api:latest，并支持回滚。
#
# 用法：
#   # 1) 显式指定版本（推荐生产用法，日期化不可变 tag）
#   ./deploy/build_production_image.sh 2026.07.15-1
#   VERSION=2026.07.15-1 ./deploy/build_production_image.sh
#
#   # 2) 不传版本时，自动用 git describe/short SHA 生成（适合本地/调试）
#   ./deploy/build_production_image.sh
#
#   # 3) 自定义镜像仓库前缀
#   IMAGE_REPO=registry.example.com/longdao/sub2api ./deploy/build_production_image.sh 2026.07.15-1
#
#   # 4) 工作区有未提交改动时强制构建（不推荐，镜像无法精确映射 commit）
#   ALLOW_DIRTY=1 ./deploy/build_production_image.sh 2026.07.15-1
#
# 版本 tag 命名建议：<YYYY.MM.DD>-<当日序号>，例如 2026.07.15-1、2026.07.15-2。
# 便于按时间排序、肉眼识别构建日期，且保证不可变（同一 tag 不重复构建覆盖）。
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 镜像仓库前缀，可通过环境变量覆盖
IMAGE_REPO="${IMAGE_REPO:-longdao/sub2api}"

# 国内构建加速参数（保留自 build_image.sh）
GOPROXY_VALUE="https://goproxy.cn,direct"
GOSUMDB_VALUE="sum.golang.google.cn"

log()  { printf '\033[1;34m[build]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

cd "${REPO_ROOT}"

# -----------------------------------------------------------------------------
# 1. 校验 git 工作区状态（REL-002：基于已提交 commit 构建）
# -----------------------------------------------------------------------------
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    err "当前目录不是 git 仓库，无法保证镜像可映射到 commit。"
    exit 1
fi

DIRTY=""
if [ -n "$(git status --porcelain)" ]; then
    DIRTY="1"
fi

if [ -n "${DIRTY}" ]; then
    warn "=============================================================="
    warn " 检测到未提交的改动（git 工作区 dirty）！"
    warn " 此时构建的镜像无法精确映射到某个干净的 git commit，"
    warn " 违反 REL-002（基于已提交 commit 构建）。"
    warn " 请先 commit 后再构建；仅调试可用 ALLOW_DIRTY=1 跳过此检查。"
    warn "=============================================================="
    git status --short >&2 || true
    if [ "${ALLOW_DIRTY:-0}" != "1" ]; then
        err "工作区不干净，已终止。设置 ALLOW_DIRTY=1 可强制构建（不推荐）。"
        exit 1
    fi
    warn "ALLOW_DIRTY=1 已设置，继续构建（镜像 commit 映射不精确）。"
fi

# -----------------------------------------------------------------------------
# 2. 解析版本号
#    优先级：命令行参数 / VERSION 环境变量 > git describe/short SHA > VERSION 文件
# -----------------------------------------------------------------------------
VERSION="${1:-${VERSION:-}}"

if [ -z "${VERSION}" ]; then
    if GIT_VERSION="$(git describe --tags --always --dirty 2>/dev/null)" && [ -n "${GIT_VERSION}" ]; then
        VERSION="${GIT_VERSION}"
    elif GIT_SHORT="$(git rev-parse --short HEAD 2>/dev/null)" && [ -n "${GIT_SHORT}" ]; then
        VERSION="${GIT_SHORT}"
    fi
fi

if [ -z "${VERSION}" ]; then
    VERSION_FILE="${REPO_ROOT}/backend/cmd/server/VERSION"
    if [ -f "${VERSION_FILE}" ]; then
        VERSION="$(tr -d '\r\n' < "${VERSION_FILE}")"
    fi
fi

if [ -z "${VERSION}" ]; then
    err "无法确定版本号（参数/git/VERSION 文件均为空）。"
    exit 1
fi

# -----------------------------------------------------------------------------
# 3. 采集 commit / 构建时间
# -----------------------------------------------------------------------------
COMMIT="$(git rev-parse HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

IMAGE_VERSIONED="${IMAGE_REPO}:${VERSION}"
IMAGE_LATEST="${IMAGE_REPO}:latest"

log "仓库根目录 : ${REPO_ROOT}"
log "镜像仓库   : ${IMAGE_REPO}"
log "版本 tag   : ${VERSION}"
log "commit SHA : ${COMMIT}"
log "构建时间   : ${DATE}"
log "目标镜像   : ${IMAGE_VERSIONED}  (+ ${IMAGE_LATEST})"

# -----------------------------------------------------------------------------
# 4. 构建
# -----------------------------------------------------------------------------
docker build \
    -t "${IMAGE_VERSIONED}" \
    -t "${IMAGE_LATEST}" \
    --build-arg "VERSION=${VERSION}" \
    --build-arg "COMMIT=${COMMIT}" \
    --build-arg "DATE=${DATE}" \
    --build-arg "GOPROXY=${GOPROXY_VALUE}" \
    --build-arg "GOSUMDB=${GOSUMDB_VALUE}" \
    -f "${REPO_ROOT}/Dockerfile" \
    "${REPO_ROOT}"

# -----------------------------------------------------------------------------
# 5. 构建结果与提示
# -----------------------------------------------------------------------------
log "=============================================================="
log " 构建成功"
log "   镜像       : ${IMAGE_VERSIONED}"
log "   便利 tag   : ${IMAGE_LATEST}"
log "   version    : ${VERSION}"
log "   commit SHA : ${COMMIT}"
log ""
log " 查看镜像 digest（用于唯一标识/推送校验）："
log "   docker images --digests ${IMAGE_REPO}"
log ""
log " 生产部署：请将 deploy/docker-compose.yml 的 image 字段指向带版本的 tag："
log "   image: ${IMAGE_VERSIONED}"
log ""
log " 回滚：把 compose 的 image 换回上一个版本 tag（例如上一次的 ${IMAGE_REPO}:<旧版本>）"
log "        然后 docker compose up -d 即可。切勿依赖 latest 做生产回滚。"
log "=============================================================="
