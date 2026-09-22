#!/bin/sh
# Generate a persistent KEK and first admin password when the operator
# did not supply them. Secrets live next to the SQLite file so a volume
# restore stays consistent.
set -eu

DATA_DIR=/var/lib/fenghuolun
mkdir -p "$DATA_DIR"

rand_b64() {
	head -c "$1" /dev/urandom | base64 | tr -d '\n'
}

rand_pass() {
	head -c 12 /dev/urandom | base64 | tr -d '/+=\n'
}

if [ -z "${FENGHUOLUN_TOKEN_KEK:-}" ]; then
	if [ -f "$DATA_DIR/token.kek" ]; then
		FENGHUOLUN_TOKEN_KEK=$(cat "$DATA_DIR/token.kek")
	else
		FENGHUOLUN_TOKEN_KEK=$(rand_b64 48)
		umask 077
		printf '%s' "$FENGHUOLUN_TOKEN_KEK" >"$DATA_DIR/token.kek"
		echo "[fenghuolun] 已生成 FENGHUOLUN_TOKEN_KEK -> $DATA_DIR/token.kek"
		echo "[fenghuolun] 请与数据库一起备份；丢失后旧绑定无法解密，只能重新绑定"
	fi
	export FENGHUOLUN_TOKEN_KEK
fi

if [ -z "${FENGHUOLUN_ADMIN_BOOTSTRAP_USER:-}" ]; then
	FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin
	export FENGHUOLUN_ADMIN_BOOTSTRAP_USER
fi

if [ -z "${FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD:-}" ]; then
	BOOTSTRAP_FILE="$DATA_DIR/admin-bootstrap.txt"
	if [ ! -f "$BOOTSTRAP_FILE" ]; then
		PASS=$(rand_pass)
		FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD=$PASS
		export FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD
		umask 077
		cat >"$BOOTSTRAP_FILE" <<EOF
username: ${FENGHUOLUN_ADMIN_BOOTSTRAP_USER}
password: ${PASS}
EOF
		echo "[fenghuolun] 首次管理员  用户=${FENGHUOLUN_ADMIN_BOOTSTRAP_USER}  密码=${PASS}"
		echo "[fenghuolun] 也写在 $BOOTSTRAP_FILE ，登录后请立刻改密"
	fi
fi

# 后台一点更新把新版本放在数据库所在目录。容器重启后跑这一份，不依赖 Docker 套接字。
APP_ROOT="$DATA_DIR"
if [ -n "${FENGHUOLUN_SQLITE_PATH:-}" ] && [ "$FENGHUOLUN_SQLITE_PATH" != ":memory:" ]; then
	APP_ROOT=$(dirname "$FENGHUOLUN_SQLITE_PATH")
fi
APP_DIR="$APP_ROOT/app/current"
if [ -x "$APP_DIR/fenghuolun" ]; then
	if [ -d "$APP_DIR/admin" ]; then
		FENGHUOLUN_ADMIN_DIR="$APP_DIR/admin"
		export FENGHUOLUN_ADMIN_DIR
	fi
	if [ -d "$APP_DIR/owner" ]; then
		FENGHUOLUN_OWNER_DIR="$APP_DIR/owner"
		export FENGHUOLUN_OWNER_DIR
	fi
	exec "$APP_DIR/fenghuolun" "$@"
fi

exec /app/fenghuolun "$@"
