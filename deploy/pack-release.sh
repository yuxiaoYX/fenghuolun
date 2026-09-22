#!/usr/bin/env bash
# 打 GitHub Release 程序包：二进制 + 管理端 + 车主页。后台一点更新下载的就是这个。
set -euo pipefail

TAG="${1:?tag, e.g. v0.1.4}"
SHA="${2:-}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist/release"
ADMIN="$ROOT/apps/admin/dist"
OWNER="$ROOT/apps/owner/h5-dist"

if [[ ! -f "$ADMIN/index.html" ]]; then
	echo "missing $ADMIN/index.html — run pnpm --dir apps/admin build" >&2
	exit 1
fi
if [[ ! -f "$OWNER/index.html" ]]; then
	echo "missing $OWNER/index.html — run pnpm owner:build-h5" >&2
	exit 1
fi

rm -rf "$OUT"
mkdir -p "$OUT"

pack_arch() {
	local arch="$1"
	local stage
	stage="$(mktemp -d)"
	CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -C "$ROOT/server" -trimpath \
		-ldflags="-s -w -X fenghuolun/internal/version.Version=${TAG} -X fenghuolun/internal/version.Commit=${SHA}" \
		-o "$stage/fenghuolun" .
	chmod 0755 "$stage/fenghuolun"
	mkdir -p "$stage/admin" "$stage/owner"
	cp -a "$ADMIN/." "$stage/admin/"
	cp -a "$OWNER/." "$stage/owner/"
	local name="fenghuolun_${TAG}_linux_${arch}.tar.gz"
	tar -C "$stage" -czf "$OUT/$name" fenghuolun admin owner
	rm -rf "$stage"
}

pack_arch amd64
pack_arch arm64

(
	cd "$OUT"
	sha256sum fenghuolun_"${TAG}"_linux_*.tar.gz > SHA256SUMS
)
echo "packed $OUT"