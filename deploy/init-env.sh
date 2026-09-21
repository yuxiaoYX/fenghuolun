#!/usr/bin/env bash
# 在仓库根目录生成 .env（密钥 + 首次管理员密码）。已有 .env 时不覆盖。
#   ./deploy/init-env.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  echo "[OK] 已有 $ROOT/.env ，不覆盖"
  exit 0
fi

if [[ ! -f .env.example ]]; then
  echo "[ERR] 找不到 .env.example" >&2
  exit 1
fi

rand_b64() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 "$1" | tr -d '\n'
  else
    head -c "$1" /dev/urandom | base64 | tr -d '\n'
  fi
}

rand_pass() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
  else
    head -c 12 /dev/urandom | base64 | tr -d '/+=\n'
  fi
}

KEK=$(rand_b64 48)
PASS=$(rand_pass)

umask 077
cat >.env <<EOF
IMAGE_TAG=latest
FENGHUOLUN_PUBLISH=8088:8088
FENGHUOLUN_DATA_DIR=./data
FENGHUOLUN_TOKEN_KEK=${KEK}
FENGHUOLUN_CORS_ORIGINS=
FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin
FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD=${PASS}
EOF

echo "[OK] 已写入 $ROOT/.env"
echo
echo "----------------------------------------"
echo "  首次管理员"
echo "  用户: admin"
echo "  密码: ${PASS}"
echo "----------------------------------------"
echo
echo "有域名请编辑 .env 的 FENGHUOLUN_CORS_ORIGINS。"
echo "登录改密后，可从 .env 删除 BOOTSTRAP 两行。"
echo "然后："
echo "  docker compose up -d"
