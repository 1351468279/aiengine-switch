#!/bin/sh
set -eu

ASSET_BRANCH=setup-assets

if [ "$#" -ne 1 ]; then
  printf '用法: %s setup-v1.0.0\n' "$0" >&2
  exit 2
fi

tag=$1
printf '%s\n' "$tag" | grep -Eq '^setup-v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)*$' || {
  printf '无效的发布标签: %s\n' "$tag" >&2
  exit 2
}

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tag_commit=$(git -C "$repo_root" rev-parse --verify "$tag^{commit}") || {
  printf '找不到标签: %s\n' "$tag" >&2
  exit 1
}
origin_url=$(git -C "$repo_root" remote get-url origin)
author_name=$(git -C "$repo_root" config user.name)
author_email=$(git -C "$repo_root" config user.email)
publish_tmp=$(mktemp -d /tmp/aiengine-assets.XXXXXX)
source_tree="$publish_tmp/source"
asset_repo="$publish_tmp/repository"
dist_dir="$publish_tmp/dist"

cleanup() {
  if [ -d "$source_tree" ]; then
    git -C "$repo_root" worktree remove --force "$source_tree" >/dev/null 2>&1 || true
  fi
  if [ -d "$publish_tmp" ]; then
    rm -rf -- "$publish_tmp"
  fi
}
trap cleanup EXIT HUP INT TERM

git -C "$repo_root" worktree add --detach "$source_tree" "$tag_commit"
(
  cd "$source_tree"
  ./scripts/build-release.sh "$tag" "$dist_dir"
)

git clone --no-checkout "$origin_url" "$asset_repo"
git -C "$asset_repo" config user.name "$author_name"
git -C "$asset_repo" config user.email "$author_email"
if git -C "$asset_repo" ls-remote --exit-code --heads origin "$ASSET_BRANCH" >/dev/null 2>&1; then
  git -C "$asset_repo" checkout -B "$ASSET_BRANCH" "origin/$ASSET_BRANCH"
  git -C "$asset_repo" rm -rf --ignore-unmatch .
else
  git -C "$asset_repo" checkout --orphan "$ASSET_BRANCH"
  git -C "$asset_repo" rm -rf --ignore-unmatch .
fi

for asset in "$dist_dir"/aiengine-setup_*.tar.gz "$dist_dir"/aiengine-setup_*.zip \
  "$dist_dir"/CHECKSUMS.txt "$dist_dir"/latest.json "$dist_dir"/install.sh "$dist_dir"/install.ps1; do
  install -m 0644 "$asset" "$asset_repo/${asset##*/}"
done

git -C "$asset_repo" add .
if git -C "$asset_repo" diff --cached --quiet; then
  printf 'GitHub 备用资产已经是 %s，无需更新。\n' "$tag"
  exit 0
fi
git -C "$asset_repo" commit -m "publish: $tag"
git -C "$asset_repo" push origin "$ASSET_BRANCH"
printf 'GitHub 备用资产已发布: %s (%s)\n' "$ASSET_BRANCH" "$tag"
