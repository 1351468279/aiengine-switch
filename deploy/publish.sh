#!/bin/sh
set -eu

REPOSITORY="1351468279/aiengine-switch"
WEBROOT="/www/wwwroot/newapi.aiare.cloud/aiengine-setup"
NGINX_TARGET="/www/server/panel/vhost/nginx/extension/newapi.aiare.cloud/aiengine-setup.conf"
LEGACY_NGINX_TARGET="/www/server/panel/vhost/nginx/extension/newapi.aiare.cloud/aiare-setup.conf"

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
asset_branch_url="https://raw.githubusercontent.com/$REPOSITORY/setup-assets"
publish_tmp=$(mktemp -d /tmp/aiengine-publish.XXXXXX)
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

download_source() {
  source_base=$1
  rm -f -- "$publish_tmp/CHECKSUMS.txt" "$publish_tmp/latest.json"
  curl -fL --retry 3 --connect-timeout 10 "$source_base/CHECKSUMS.txt" -o "$publish_tmp/CHECKSUMS.txt" || return 1
  if grep -Eq '[[:space:]]aiengine-setup_linux_amd64\.tar\.gz$' "$publish_tmp/CHECKSUMS.txt"; then
    asset_prefix="aiengine-setup"
  elif grep -Eq '[[:space:]]aiare-setup_linux_amd64\.tar\.gz$' "$publish_tmp/CHECKSUMS.txt"; then
    # setup-v1.1.0 and older releases use the pre-rename asset prefix.
    asset_prefix="aiare-setup"
  else
    return 1
  fi
  assets="
${asset_prefix}_linux_amd64.tar.gz
${asset_prefix}_linux_arm64.tar.gz
${asset_prefix}_darwin_amd64.tar.gz
${asset_prefix}_darwin_arm64.tar.gz
${asset_prefix}_windows_amd64.zip
${asset_prefix}_windows_arm64.zip
latest.json
install.sh
install.ps1
"
  for asset in $assets; do
    curl -fL --retry 3 --connect-timeout 10 "$source_base/$asset" -o "$publish_tmp/$asset" || return 1
  done
  (
    cd "$publish_tmp"
    sha256sum -c CHECKSUMS.txt
  ) || return 1
}

printf '下载发布资产 %s...\n' "$tag"
if download_source "$release_url"; then
  printf '发布源: GitHub Release\n'
elif download_source "$asset_branch_url"; then
  branch_tag=$(sed -n 's/.*"tag"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$publish_tmp/latest.json")
  if [ "$branch_tag" != "$tag" ]; then
    printf 'GitHub 备用源版本不匹配: 需要 %s，实际 %s\n' "$tag" "${branch_tag:-未知}" >&2
    exit 1
  fi
  printf '发布源: GitHub setup-assets 备用分支\n'
else
  printf 'GitHub Release 和 setup-assets 备用分支均下载或校验失败。\n' >&2
  exit 1
fi

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
  for asset in CHECKSUMS.txt $assets; do
    install -m 0644 "$publish_tmp/$asset" "$stage/$asset"
  done
  mv "$stage" "$target"
fi
ln -s "releases/$tag" "$WEBROOT/.current.$$"
mv -Tf "$WEBROOT/.current.$$" "$WEBROOT/current"

nginx_backup="$publish_tmp/nginx.previous"
legacy_nginx_backup="$publish_tmp/nginx.legacy.previous"
nginx_existed=0
legacy_nginx_existed=0
if [ -f "$NGINX_TARGET" ]; then
  cp "$NGINX_TARGET" "$nginx_backup"
  nginx_existed=1
fi
if [ -f "$LEGACY_NGINX_TARGET" ]; then
  cp "$LEGACY_NGINX_TARGET" "$legacy_nginx_backup"
  legacy_nginx_existed=1
fi
restore_nginx() {
  if [ "$nginx_existed" -eq 1 ]; then
    cp "$nginx_backup" "$NGINX_TARGET"
  else
    rm -f -- "$NGINX_TARGET"
  fi
  if [ "$legacy_nginx_existed" -eq 1 ]; then
    cp "$legacy_nginx_backup" "$LEGACY_NGINX_TARGET"
  else
    rm -f -- "$LEGACY_NGINX_TARGET"
  fi
}
install -m 0644 "$script_dir/nginx-aiengine-setup.conf" "$NGINX_TARGET"
rm -f -- "$LEGACY_NGINX_TARGET"
if ! nginx -t; then
  restore_nginx
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
  restore_nginx
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
