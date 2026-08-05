#!/bin/sh
set -eu

REPOSITORY="1351468279/aiengine-switch"
WEBROOT="/www/wwwroot/newapi.aiare.cloud/aiare-setup"
NGINX_TARGET="/www/server/panel/vhost/nginx/extension/newapi.aiare.cloud/aiare-setup.conf"

if [ "$#" -ne 1 ]; then
  printf '用法: %s setup-v1.0.0\n' "$0" >&2
  exit 2
fi
tag=$1
printf '%s\n' "$tag" | grep -Eq '^setup-v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)*$' || {
  printf '无效的发布标签: %s\n' "$tag" >&2
  exit 2
}

if [ "$(id -u)" -ne 0 ]; then
  printf '请以 root 身份运行发布脚本。\n' >&2
  exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
release_url="https://github.com/$REPOSITORY/releases/download/$tag"
publish_tmp=$(mktemp -d /tmp/aiare-publish.XXXXXX)
stage="$WEBROOT/releases/.${tag}.stage.$$"
target="$WEBROOT/releases/$tag"
old_current=""
if [ -L "$WEBROOT/current" ]; then
  old_current=$(readlink "$WEBROOT/current")
fi

cleanup() {
  if [ -n "${publish_tmp:-}" ] && [ -d "$publish_tmp" ]; then
    rm -rf -- "$publish_tmp"
  fi
  if [ -n "${stage:-}" ] && [ -d "$stage" ]; then
    rm -rf -- "$stage"
  fi
}
trap cleanup EXIT HUP INT TERM

assets="
aiare-setup_linux_amd64.tar.gz
aiare-setup_linux_arm64.tar.gz
aiare-setup_darwin_amd64.tar.gz
aiare-setup_darwin_arm64.tar.gz
aiare-setup_windows_amd64.zip
aiare-setup_windows_arm64.zip
CHECKSUMS.txt
latest.json
install.sh
install.ps1
"

printf '下载 GitHub Release %s...\n' "$tag"
for asset in $assets; do
  curl -fL --retry 3 --connect-timeout 10 "$release_url/$asset" -o "$publish_tmp/$asset"
done
(
  cd "$publish_tmp"
  sha256sum -c CHECKSUMS.txt
)

mkdir -p "$WEBROOT/releases"
if [ -e "$target" ]; then
  if ! cmp -s "$publish_tmp/CHECKSUMS.txt" "$target/CHECKSUMS.txt"; then
    printf '目标发布目录已存在但校验清单不同，停止以避免覆盖: %s\n' "$target" >&2
    exit 1
  fi
  (
    cd "$target"
    sha256sum -c CHECKSUMS.txt
  )
  printf '复用已校验的发布目录: %s\n' "$target"
else
  mkdir "$stage"
  for asset in $assets; do
    install -m 0644 "$publish_tmp/$asset" "$stage/$asset"
  done
  mv "$stage" "$target"
fi
ln -s "releases/$tag" "$WEBROOT/.current.$$"
mv -Tf "$WEBROOT/.current.$$" "$WEBROOT/current"

nginx_backup="$publish_tmp/nginx.previous"
nginx_existed=0
if [ -f "$NGINX_TARGET" ]; then
  cp "$NGINX_TARGET" "$nginx_backup"
  nginx_existed=1
fi
install -m 0644 "$script_dir/nginx-aiare-setup.conf" "$NGINX_TARGET"
if ! nginx -t; then
  if [ "$nginx_existed" -eq 1 ]; then
    cp "$nginx_backup" "$NGINX_TARGET"
  else
    rm -f -- "$NGINX_TARGET"
  fi
  if [ -n "$old_current" ]; then
    ln -s "$old_current" "$WEBROOT/.current.rollback.$$"
    mv -Tf "$WEBROOT/.current.rollback.$$" "$WEBROOT/current"
  else
    rm -f -- "$WEBROOT/current"
  fi
  printf 'Nginx 校验失败，已恢复之前的入口。\n' >&2
  exit 1
fi
if ! nginx -s reload; then
  if [ "$nginx_existed" -eq 1 ]; then
    cp "$nginx_backup" "$NGINX_TARGET"
  else
    rm -f -- "$NGINX_TARGET"
  fi
  if [ -n "$old_current" ]; then
    ln -s "$old_current" "$WEBROOT/.current.rollback.$$"
    mv -Tf "$WEBROOT/.current.rollback.$$" "$WEBROOT/current"
  else
    rm -f -- "$WEBROOT/current"
  fi
  nginx -t && nginx -s reload || true
  printf 'Nginx 重载失败，已恢复之前的入口。\n' >&2
  exit 1
fi
printf '发布完成: https://modelapi.aiaiaiaiai.cloud/install.sh (%s)\n' "$tag"
