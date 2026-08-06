#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
  printf '用法: %s setup-v1.0.0 输出目录\n' "$0" >&2
  exit 2
fi

release_tag=$1
output_dir=$2
printf '%s\n' "$release_tag" | grep -Eq '^setup-v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)*$' || {
  printf '无效的发布标签: %s\n' "$release_tag" >&2
  exit 2
}

release_version=${release_tag#setup-v}
mkdir -p "$output_dir/work"

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  setup_os=${target%/*}
  setup_arch=${target#*/}
  package_name="aiengine-setup_${setup_os}_${setup_arch}"
  binary_name=aiengine-setup
  if [ "$setup_os" = windows ]; then
    binary_name=aiengine-setup.exe
  fi

  package_dir="$output_dir/work/$package_name"
  mkdir -p "$package_dir"
  CGO_ENABLED=0 GOOS="$setup_os" GOARCH="$setup_arch" \
    go build -trimpath -ldflags "-s -w -X main.version=$release_version" \
    -o "$package_dir/$binary_name" ./cmd/aiengine-setup

  if [ "$setup_os" = windows ]; then
    (
      cd "$package_dir"
      zip -q "../../$package_name.zip" "$binary_name"
    )
  else
    tar -C "$package_dir" -czf "$output_dir/$package_name.tar.gz" "$binary_name"
  fi
done

cp scripts/install.sh scripts/install.ps1 "$output_dir/"
(
  cd "$output_dir"
  sha256sum aiengine-setup_*.tar.gz aiengine-setup_*.zip install.sh install.ps1 > CHECKSUMS.txt
  jq -n \
    --arg version "$release_version" \
    --arg tag "$release_tag" \
    --arg published_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{schema: 1, version: $version, tag: $tag, published_at: $published_at}' > latest.json
  sha256sum -c CHECKSUMS.txt
)
