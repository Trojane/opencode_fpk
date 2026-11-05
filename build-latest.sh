#!/usr/bin/env bash
# ============================================================================
# opencode → fnOS FPK 一键重打包（已验证产出 2.2.5）
#
# 做法：取 lin1740/fnos-OpenCode 的 2.2.1 适配层 → 打 patches/ 里的 4 处补丁
#      → 换上游官方二进制 → 重编 Go 网关 → fnpack 打包。
#
# 用法：
#   ./build-latest.sh 2.2.5            # 双架构 all（默认）
#   ./build-latest.sh 2.2.5 x86        # 仅 x64
#   ./build-latest.sh 2.2.5 arm        # 仅 arm64
#
# 环境变量：
#   OC_MIRROR=https://ghfast.top       # GitHub 加速前缀（国内建议保留）
#   BASE_TAG=<git tag/branch>          # 适配层基线，默认 main（内含 opencode_all_2.2.1）
#
# 依赖：git curl tar fnpack go(>=1.21)
#   fnpack: https://static2.fnnas.com/fnpack/fnpack-1.2.3-linux-amd64
# ============================================================================
set -euo pipefail

VER="${1:-}"
PLATFORM="${2:-all}"
[ -n "$VER" ] || { echo "用法: $0 <opencode版本号，如 2.2.5> [all|x86|arm]" >&2; exit 1; }
case "$PLATFORM" in all|x86|arm) ;; *) echo "PLATFORM 只能是 all / x86 / arm" >&2; exit 1 ;; esac

ROOT="$(cd "$(dirname "$0")" && pwd)"
WORK="$ROOT/work"
DIST="$ROOT/dist"
PATCHES="$ROOT/patches"
MIRROR="${OC_MIRROR:-https://ghfast.top}"
BASE_REPO="https://github.com/lin1740/fnos-OpenCode.git"
BASE_TAG="${BASE_TAG:-main}"
BASE_DIR="$WORK/base"

for c in git curl tar fnpack go; do
  command -v "$c" >/dev/null 2>&1 || { echo "缺少命令: $c" >&2; exit 1; }
done

mkdir -p "$WORK" "$DIST"

# ---------------------------------------------------------------- 1. 适配层基线
if [ ! -d "$BASE_DIR/.git" ]; then
  echo "==> 拉取适配层基线 ($BASE_TAG)"
  # 国内直连 GitHub 经常断流，优先走镜像，失败再回退直连
  if ! git clone --depth 1 --branch "$BASE_TAG" "$MIRROR/$BASE_REPO" "$BASE_DIR" 2>/dev/null; then
    rm -rf "$BASE_DIR"
    git clone --depth 1 --branch "$BASE_TAG" "$BASE_REPO" "$BASE_DIR"
  fi
fi

SRC_PKG="$(find "$BASE_DIR" -maxdepth 1 -type d -name 'opencode_all_*' | sort -V | tail -1)"
[ -n "$SRC_PKG" ] || { echo "基线仓库里没找到 opencode_all_* 目录" >&2; exit 1; }

PKG="$WORK/opencode"
rm -rf "$PKG"
cp -a "$SRC_PKG" "$PKG"
echo "==> 基线: $(basename "$SRC_PKG")"

# ---------------------------------------------------------------- 2. 打补丁
# patches 针对 2.2.1 基线；上游适配层若已自行修好，--forward 会跳过而不报错
echo "==> 应用适配补丁"
for p in "$PATCHES"/*.diff; do
  [ -f "$p" ] || continue
  if patch -p1 -d "$PKG" --forward --silent < "$p"; then
    echo "    ok  $(basename "$p")"
  else
    echo "    !!  $(basename "$p") 未应用（可能基线已包含），请人工确认" >&2
  fi
done

# 补丁有效性校验（避免补丁静默跳过导致打出旧逻辑）
grep -q 'PREFIX+"/gateway/status"' "$PKG/app/server/gateway.go" || {
  echo "错误：gateway.go 缺少前缀路由补丁，构建中止" >&2; exit 1; }
grep -q 'Select folder' "$PKG/app/server/shim_gen.go" || {
  echo "错误：shim_gen.go 缺少一键打开项目补丁，构建中止" >&2; exit 1; }

# VERSION 常量必须与本次版本一致（补丁里写死 2.2.5，这里再兜一次底）
sed -i "s/^const VERSION = \".*\"$/const VERSION = \"$VER\"/" "$PKG/app/server/gateway.go"

# ---------------------------------------------------------------- 3. 官方二进制
fetch_arch() {
  local a="$1" dst="app/opencode-linux-$1"
  local url="$MIRROR/https://github.com/anomalyco/opencode/releases/download/v$VER/opencode-linux-$a.tar.gz"
  rm -rf "$WORK/oc-$a"; mkdir -p "$WORK/oc-$a"
  if [ -s "$WORK/oc-$a.tar.gz" ]; then
    echo "==> 复用缓存 opencode v$VER linux-$a"
  else
    echo "==> 下载 opencode v$VER linux-$a"
    curl -fL --retry 3 --max-time 1800 -o "$WORK/oc-$a.tar.gz" "$url"
  fi
  tar -xzf "$WORK/oc-$a.tar.gz" -C "$WORK/oc-$a"
  [ -f "$WORK/oc-$a/opencode" ] || { echo "压缩包里没有 opencode 可执行文件" >&2; exit 1; }
  install -m 0755 "$WORK/oc-$a/opencode" "$PKG/$dst"
}

case "$PLATFORM" in
  all) fetch_arch x64; fetch_arch arm64 ;;
  x86) fetch_arch x64; rm -f "$PKG/app/opencode-linux-arm64" "$PKG/app/server/gateway_arm64" ;;
  arm) fetch_arch arm64; rm -f "$PKG/app/opencode-linux-x64" "$PKG/app/server/gateway_amd64" ;;
esac

# ---------------------------------------------------------------- 4. manifest
sed -i "s/^version[[:space:]]*=.*/version                    = $VER/" "$PKG/manifest"
sed -i "s/^platform[[:space:]]*=.*/platform                   = $PLATFORM/" "$PKG/manifest"
sed -i "s|^changelog[[:space:]]*=.*|changelog                  = 版本 $VER　同步上游官方 $VER，并适配 v2 服务端（/api 前缀、serve 随机密码自动注入）。|" "$PKG/manifest"

# ---------------------------------------------------------------- 5. 重编网关
echo "==> 编译 Go 网关"
(
  cd "$PKG/app/server"
  if [ -f gateway_amd64 ] || [ "$PLATFORM" = "x86" ]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o gateway_amd64 .
  fi
  if [ -f gateway_arm64 ] || [ "$PLATFORM" = "arm" ]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o gateway_arm64 .
  fi
  chmod 755 gateway_amd64 gateway_arm64 2>/dev/null || true
)

# ---------------------------------------------------------------- 6. 打包
chmod 755 "$PKG/app/opencode-linux-"* "$PKG/cmd/"* 2>/dev/null || true
echo "==> fnpack build"
( cd "$WORK" && fnpack build -d opencode >/dev/null )

OUT="$DIST/opencode_all_${VER}.fpk"
[ "$PLATFORM" != "all" ] && OUT="$DIST/opencode_all_${VER}_${PLATFORM}.fpk"
mv -f "$WORK/opencode.fpk" "$OUT"

echo "==> 完成: $OUT ($(du -h "$OUT" | cut -f1))"
sha256sum "$OUT"
