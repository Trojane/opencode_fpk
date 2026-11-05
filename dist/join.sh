#!/usr/bin/env bash
# 把 dist 下的分卷合并成可安装的 fpk，并校验 SHA-256
set -euo pipefail
cd "$(dirname "$0")"

OUT="opencode_all_2.2.5.fpk"
EXPECT="73efb29af0dd3141b0002884cadf6a8c3188a15964faac9aa87b2d0c7426d6d7"

cat opencode_all_2.2.5.fpk.part-* > "$OUT"
echo "已生成 $OUT"

ACTUAL="$(sha256sum "$OUT" | awk '{print $1}')"
if [ "$ACTUAL" = "$EXPECT" ]; then
  echo "校验通过: $ACTUAL"
else
  echo "校验失败!" >&2
  echo "  期望: $EXPECT" >&2
  echo "  实际: $ACTUAL" >&2
  exit 1
fi
