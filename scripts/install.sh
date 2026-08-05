#!/bin/sh
set -eu

PRIMARY_BASE="https://modelapi.aiaiaiaiai.cloud/aiare-setup/current"
FALLBACK_BASE="https://github.com/1351468279/aiengine-switch/releases/latest/download"

fail() {
  printf '安装失败: %s\n' "$1" >&2
  exit 1
}

download() {
  source_url=$1
  destination=$2
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 2 --connect-timeout 10 "$source_url" -o "$destination"
  elif command -v wget >/dev/null 2>&1; then
    wget -q --timeout=15 -O "$destination" "$source_url"
  else
    fail "需要 curl 或 wget"
  fi
}

sha256_file() {
  target_file=$1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$target_file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$target_file" | awk '{print $1}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$target_file" | awk '{print $NF}'
  else
    fail "需要 sha256sum、shasum 或 openssl 来校验下载文件"
  fi
}

case "$(uname -s)" in
  Linux) setup_os=linux ;;
  Darwin) setup_os=darwin ;;
  *) fail "仅支持 Linux、WSL 和 macOS；Windows 请使用 install.ps1" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) setup_arch=amd64 ;;
  arm64|aarch64) setup_arch=arm64 ;;
  *) fail "不支持的 CPU 架构: $(uname -m)" ;;
esac

archive="aiare-setup_${setup_os}_${setup_arch}.tar.gz"
setup_tmp=$(mktemp -d 2>/dev/null || mktemp -d -t aiare-setup) || fail "无法创建临时目录"
cleanup() {
  if [ -n "${setup_tmp:-}" ] && [ -d "$setup_tmp" ]; then
    rm -rf -- "$setup_tmp"
  fi
}
trap cleanup EXIT HUP INT TERM

download_release() {
  release_base=$1
  rm -f -- "$setup_tmp/$archive" "$setup_tmp/CHECKSUMS.txt"
  download "$release_base/CHECKSUMS.txt" "$setup_tmp/CHECKSUMS.txt" || return 1
  download "$release_base/$archive" "$setup_tmp/$archive" || return 1
  expected=$(awk -v name="$archive" '$2 == name || $2 == "*" name {print $1; exit}' "$setup_tmp/CHECKSUMS.txt")
  [ -n "$expected" ] || return 1
  actual=$(sha256_file "$setup_tmp/$archive")
  [ "$actual" = "$expected" ] || return 1
}

printf '正在获取适用于 %s/%s 的 AIARE 安装器...\n' "$setup_os" "$setup_arch"
if download_release "$PRIMARY_BASE"; then
  printf '下载源: AIARE\n'
elif download_release "$FALLBACK_BASE"; then
  printf '下载源: GitHub Release（AIARE 下载源不可用）\n'
else
  fail "主下载源和 GitHub Release 均下载或校验失败"
fi

mkdir "$setup_tmp/extract"
tar -xzf "$setup_tmp/$archive" -C "$setup_tmp/extract"
binary="$setup_tmp/extract/aiare-setup"
[ -f "$binary" ] || fail "发布包中缺少 aiare-setup"
chmod 700 "$binary"
"$binary" install "$@"
