#!/usr/bin/env bash
# 风火轮 1Panel / Linux 一键部署（容器只听 127.0.0.1:18088）。
#   curl -sSL https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/deploy/install.sh | bash
# 克隆仓库后用 Docker Compose 的方式见 docs/DEPLOYMENT.md。
set -euo pipefail

PREFIX=/opt/fenghuolun
DOMAIN=""
COMMAND=install
SCRIPT_URL="https://raw.githubusercontent.com/yuxiaoYX/fenghuolun/master/deploy/install.sh"
IMAGE="ghcr.io/yuxiaoyx/fenghuolun:latest"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $*"; }
ok() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err() { echo -e "${RED}[ERR]${NC} $*" >&2; }

usage() {
  cat <<'EOF'
用法:
  install.sh              安装或按现有配置重建
  install.sh upgrade      备份数据、拉取 latest、重建
  install.sh uninstall    停止容器（保留数据和密钥）
  install.sh uninstall --purge
                          停止并删除 /opt/fenghuolun（不可恢复）

选项:
  --domain https://你的域名   写入 CORS（仅首次生成 .env.production 时）
  --prefix DIR               安装根目录，默认 /opt/fenghuolun

克隆仓库后用 Docker Compose（不走本脚本）见 docs/DEPLOYMENT.md:
  git clone https://github.com/yuxiaoYX/fenghuolun.git
  cd fenghuolun && bash deploy/init-env.sh && docker compose up -d
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    install | upgrade | uninstall) COMMAND=$1 ;;
    --purge) PURGE=1 ;;
    --domain)
      DOMAIN=${2:-}
      shift
      ;;
    --prefix)
      PREFIX=${2:-}
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      err "未知参数: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

PURGE=${PURGE:-0}
DEPLOY_DIR="$PREFIX/deploy"
DATA_DIR="$PREFIX/data"
BACKUP_DIR="$PREFIX/backups"
ENV_FILE="$DEPLOY_DIR/.env.production"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    err "找不到命令: $1"
    exit 1
  }
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$DEPLOY_DIR/docker-compose.yml" --project-directory "$DEPLOY_DIR" "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "$DEPLOY_DIR/docker-compose.yml" --project-directory "$DEPLOY_DIR" "$@"
  else
    err "需要 Docker Compose v2（docker compose）"
    exit 1
  fi
}

write_compose() {
  cat >"$DEPLOY_DIR/docker-compose.yml" <<'EOF'
name: fenghuolun

services:
  fenghuolun:
    image: ghcr.io/yuxiaoyx/fenghuolun:${IMAGE_TAG:-latest}
    container_name: fenghuolun
    restart: unless-stopped
    env_file:
      - .env.production
    environment:
      FENGHUOLUN_HTTP_ADDR: 0.0.0.0:8088
      FENGHUOLUN_SQLITE_PATH: /var/lib/fenghuolun/fenghuolun.db
      FENGHUOLUN_ADMIN_DIR: /app/admin
    ports:
      - "127.0.0.1:18088:8088"
    volumes:
      - /opt/fenghuolun/data:/var/lib/fenghuolun
      - /var/run/docker.sock:/var/run/docker.sock
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8088/healthz"]
      interval: 15s
      timeout: 3s
      retries: 10
      start_period: 20s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
EOF
  if [[ "$PREFIX" != "/opt/fenghuolun" ]]; then
    if sed --version >/dev/null 2>&1; then
      sed -i "s#/opt/fenghuolun/data#${DATA_DIR}#g" "$DEPLOY_DIR/docker-compose.yml"
    else
      sed -i '' "s#/opt/fenghuolun/data#${DATA_DIR}#g" "$DEPLOY_DIR/docker-compose.yml"
    fi
  fi
}

write_upgrade_helper() {
  cat >"$DEPLOY_DIR/upgrade.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
cd "$DEPLOY_DIR"
mkdir -p "$BACKUP_DIR"
if [[ -d "$DATA_DIR" ]] && [[ -n "\$(ls -A "$DATA_DIR" 2>/dev/null || true)" ]]; then
  ts=\$(date +%Y%m%d%H%M%S)
  cp -a "$DATA_DIR" "$BACKUP_DIR/pre-update-\$ts"
fi
if docker compose version >/dev/null 2>&1; then
  docker compose pull
  docker compose up -d --remove-orphans
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose pull
  docker-compose up -d --remove-orphans
else
  echo "需要 docker compose" >&2
  exit 1
fi
curl -fsS http://127.0.0.1:18088/healthz | grep -q phase
echo "upgrade ok"
EOF
  chmod +x "$DEPLOY_DIR/upgrade.sh" 2>/dev/null || true
}

install_self() {
  local dest="$DEPLOY_DIR/install.sh"
  if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
    cp -f "${BASH_SOURCE[0]}" "$dest"
  else
    curl -fsSL "$SCRIPT_URL" -o "$dest" 2>/dev/null || true
  fi
  if [[ -f "$dest" ]]; then
    chmod +x "$dest" 2>/dev/null || true
  fi
  write_upgrade_helper
}

write_env() {
  if [[ -f "$ENV_FILE" ]]; then
    info "已有 $ENV_FILE ，不覆盖密钥"
    return
  fi
  need_cmd openssl
  local kek password
  kek=$(openssl rand -base64 48 | tr -d '\n')
  password=$(openssl rand -hex 16)
  local cors=""
  if [[ -n "$DOMAIN" ]]; then
    cors=$DOMAIN
  fi
  cat >"$ENV_FILE" <<EOF
FENGHUOLUN_TOKEN_KEK=${kek}
FENGHUOLUN_CORS_ORIGINS=${cors}
FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin
FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD=${password}
EOF
  chmod 600 "$ENV_FILE"
  umask 077
  cat >"$DEPLOY_DIR/admin-bootstrap.txt" <<EOF
用户: admin
密码: ${password}
登录后请立刻改密，然后从 .env.production 删除 BOOTSTRAP 两行，再执行:
  $DEPLOY_DIR/install.sh
EOF
  chmod 600 "$DEPLOY_DIR/admin-bootstrap.txt"
  echo ""
  echo "----------------------------------------"
  echo "  首次管理员账号（只显示这一次）"
  echo "  用户: admin"
  echo "  密码: ${password}"
  echo "  也写在: $DEPLOY_DIR/admin-bootstrap.txt"
  echo "----------------------------------------"
  echo ""
  warn "登录改密后，删掉 .env.production 里的 BOOTSTRAP 两行，再跑一次 install.sh"
}

backup_data() {
  mkdir -p "$BACKUP_DIR"
  if [[ -d "$DATA_DIR" ]] && [[ -n "$(ls -A "$DATA_DIR" 2>/dev/null || true)" ]]; then
    local ts
    ts=$(date +%Y%m%d%H%M%S)
    cp -a "$DATA_DIR" "$BACKUP_DIR/pre-update-$ts"
    info "已备份数据到 $BACKUP_DIR/pre-update-$ts"
  fi
}

wait_health() {
  local i
  for i in $(seq 1 30); do
    if curl -fsS http://127.0.0.1:18088/healthz 2>/dev/null | grep -q phase; then
      ok "healthz 正常"
      curl -fsS http://127.0.0.1:18088/healthz || true
      echo
      return 0
    fi
    sleep 1
  done
  err "容器已启动，但 127.0.0.1:18088/healthz 无响应。看日志: docker logs fenghuolun"
  exit 1
}

pull_image() {
  info "拉取 $IMAGE"
  if compose pull; then
    ok "镜像已更新"
    return
  fi
  if docker image inspect "$IMAGE" >/dev/null 2>&1; then
    warn "在线拉取失败，使用本机已有镜像"
    return
  fi
  err "拉不下 $IMAGE"
  echo "若包是私有的，先在 1Panel「容器 → 仓库」登录 ghcr.io，或："
  echo "  docker login ghcr.io -u yuxiaoYX"
  echo "权限只要 read:packages。不要把 PAT 写进 .env.production。"
  exit 1
}

start_stack() {
  pull_image
  info "启动容器"
  compose up -d --remove-orphans
  wait_health
}

cmd_install() {
  need_cmd docker
  need_cmd curl
  mkdir -p "$DEPLOY_DIR" "$DATA_DIR" "$BACKUP_DIR"
  write_compose
  write_env
  install_self
  start_stack
  echo ""
  ok "安装完成"
  echo "  数据:     $DATA_DIR"
  echo "  配置:     $ENV_FILE"
  echo "  车主:     http://127.0.0.1:18088/"
  echo "  管理端:   http://127.0.0.1:18088/admin/login"
  echo ""
  echo "1Panel 还要做两步（证书和域名）："
  echo "  1. 网站 → 创建网站 → 反向代理，目标 http://127.0.0.1:18088"
  echo "  2. 申请 Let's Encrypt，打开 HTTPS 跳转"
  echo ""
  echo "以后升级（发版后）："
  echo "  $DEPLOY_DIR/install.sh upgrade"
}

cmd_upgrade() {
  need_cmd docker
  if [[ ! -f "$DEPLOY_DIR/docker-compose.yml" ]]; then
    err "未安装。先跑 install。"
    exit 1
  fi
  write_compose
  install_self
  backup_data
  start_stack
  ok "已升级到 latest"
}

cmd_uninstall() {
  if [[ -f "$DEPLOY_DIR/docker-compose.yml" ]]; then
    compose down || true
  elif docker inspect fenghuolun >/dev/null 2>&1; then
    docker rm -f fenghuolun || true
  fi
  if [[ "$PURGE" == "1" ]]; then
    warn "删除 $PREFIX"
    rm -rf "$PREFIX"
    ok "已卸载并删除数据"
  else
    ok "已停止容器。数据仍在 $DATA_DIR ，密钥在 $ENV_FILE"
  fi
}

echo ""
echo "=========================================="
echo "  风火轮  $COMMAND"
echo "=========================================="
echo ""

case "$COMMAND" in
  install) cmd_install ;;
  upgrade) cmd_upgrade ;;
  uninstall) cmd_uninstall ;;
esac
