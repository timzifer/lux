#!/usr/bin/env bash
# Boundary lint: enforces the future multi-repo dependency DAG
# (see docs/repo-split.md). Each package group corresponds to one
# target repository; a group may only import from the groups listed
# in its allowlist. Run from the repo root:
#
#   ./scripts/check_boundaries.sh
#
# Uses production imports only (go list -deps ignores _test.go files),
# matching what a Go module boundary would enforce at compile time.
set -euo pipefail

MOD=github.com/timzifer/lux
TAGS=nogui

# Map an import path (relative to $MOD/) to its repo group.
group_of() {
  case "$1" in
    web/wasm|web/wasm/*) echo tools ;;
    draw|draw/*|input|input/*|anim|anim/*|a11y|a11y/*|interaction|interaction/*|\
    validation|validation/*|fonts|fonts/*|image|image/*|theme|theme/*|\
    textseg|textseg/*|hit|hit/*) echo core ;;
    text|text/*|render|render/*|gpu|gpu/*|wgpu|wgpu/*) echo render ;;
    platform|platform/*) echo platform ;;
    ui|ui/*|richtext|richtext/*|web|web/*|uitest|uitest/*) echo widgets ;;
    app|app/*|dialog|dialog/*|apptest|apptest/*|internal/loop|internal/loop/*|\
    vellum|vellum/*) echo app ;;
    examples|examples/*|cmd|cmd/*) echo tools ;;
    *) echo unknown ;;
  esac
}

# allowed <group> — groups a package group may depend on (besides itself).
allowed_of() {
  case "$1" in
    core) echo "core" ;;
    render) echo "render core" ;;
    platform) echo "platform core" ;;
    widgets) echo "widgets core render" ;;
    app) echo "app widgets core render platform" ;;
    tools) echo "tools app widgets core render platform" ;;
  esac
}

# Root directories per group (for go list package patterns).
dirs_for() {
  case "$1" in
    core) echo "./draw/... ./input/... ./anim/... ./a11y/... ./interaction/... ./validation/... ./fonts/... ./image/... ./theme/... ./textseg/... ./hit/..." ;;
    render) echo "./text/... ./render/... ./gpu/... ./wgpu/..." ;;
    platform) echo "./platform/..." ;;
    widgets) echo "./ui/... ./richtext/... ./web/css/... ./web/dom/... ./web ./uitest/..." ;;
    app) echo "./app/... ./dialog/... ./apptest/... ./internal/loop/... ./vellum/..." ;;
    tools) echo "./examples/... ./cmd/... ./web/wasm/..." ;;
  esac
}

fail=0
for grp in core render platform widgets app tools; do
  allowed=" $(allowed_of "$grp") "
  # shellcheck disable=SC2046
  deps=$(go list -deps -tags "$TAGS" $(dirs_for "$grp") 2>/dev/null | grep "^$MOD/" || true)
  for dep in $deps; do
    rel=${dep#"$MOD"/}
    dgrp=$(group_of "$rel")
    if [ "$dgrp" = unknown ]; then
      echo "FAIL: $grp depends on $rel — package not assigned to any repo group"
      fail=1
    elif [[ "$allowed" != *" $dgrp "* ]]; then
      echo "FAIL: $grp -> $dgrp ($rel) violates the repo DAG"
      fail=1
    fi
  done
done

if [ "$fail" -ne 0 ]; then
  echo "Boundary check failed. See docs/repo-split.md for the target DAG."
  exit 1
fi
echo "Boundary check OK: package groups respect the repo DAG."
