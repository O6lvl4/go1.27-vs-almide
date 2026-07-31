#!/bin/sh
# Go 1.27rc2 と almide 0.41 の全サンプルを並べて実行する
set -e
cd "$(dirname "$0")"

for d in 01-generic-methods 02-json-v2 03-uuid 04-stdlib-bits 05-concurrency; do
  echo "━━━━━━━━━━ $d ━━━━━━━━━━"
  echo "--- Go 1.27"
  go run "./$d"
  echo "--- almide"
  almide run "$d/main.almd"
  echo
done
