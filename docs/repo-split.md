# Repo Split — Migration Guide

Lux is being split into independent repositories so each layer can evolve,
release and be consumed on its own. **Phase A** (decoupling inside this repo)
is complete: the directory layout now mirrors the target repositories 1:1 and
`scripts/check_boundaries.sh` enforces the dependency DAG in CI.
This document describes **Phase B**: the actual extraction.

## Target repositories

Extraction order = dependency order (bottom-up). All modules `go 1.25`,
tagged `v0.x` (breaking changes allowed pre-1.0).

| # | Repo / module path | Contents (paths in this repo) | Depends on |
|---|---|---|---|
| 1 | `github.com/timzifer/wgpu` | `vendor_gogpu_wgpu/` (module path rewritten from `github.com/gogpu/wgpu`) | — |
| 2 | `github.com/timzifer/lux-core` | `draw`, `input`, `anim`, `a11y`, `interaction`, `validation`, `fonts`, `image`, `theme`, `textseg`, `hit` | — |
| 3 | `github.com/timzifer/lux-render` | `text`, `render`, `gpu`, `wgpu` | lux-core, wgpu |
| 4 | `github.com/timzifer/lux-platform` | `platform/` promoted to module root (backends as subpackages) | lux-core |
| 5 | `github.com/timzifer/lux` (this repo keeps the name) | `ui` + subpackages, `richtext`, `web/css`, `web/dom`, `web` (acid harness), `uitest` incl. goldens | lux-core, lux-render |
| 6 | `github.com/timzifer/lux-app` | `app`, `dialog`, `apptest`, `internal/loop`, `vellum` | lux-core, lux-render, lux-platform, lux |
| 7 | `github.com/timzifer/lux-tools` | `examples/*`, `cmd/lux-inspector`, `web/wasm` | all of the above |

DAG: `core → ∅; render → core,wgpu; platform → core; widgets → core,render;
app → core,render,platform,widgets; tools → all`. No cycles.

Placement notes:

- **richtext** lives with the widgets: it imports `ui` and 4 ui subpackages;
  its text needs are covered by `textseg` (lux-core) since phase A4.
- **uitest** lives with the widgets because `ui/chart` and `ui/osk` golden
  tests import it; its `render` import is why the widgets repo requires
  lux-render.
- **internal/loop** stays internal to lux-app (only app + apptest use it).
- **vellum** is exported from lux-app because `cmd/lux-inspector`
  (lux-tools) needs the client + wire format.

## Phase A recap (done in this repo)

- **A1** `ui.SetSendFunc`/`ui.Send` hook (`ui/send.go`) — `ui/data` no longer
  imports `app`.
- **A2** `ui.DialogKind` (`ui/dialog_kind.go`) — `ui` no longer imports
  `platform`; the top-level `dialog` package maps `ui.DialogKind` →
  `platform.DialogKind`.
- **A3** `platform/web` exposes `WASMCanvas()`; the app layer probes it and
  calls `wgpu.SetWASMCanvas` (`app/wasm_canvas_js.go`) — the platform layer
  has no renderer dependency.
- **A4** `textseg/` extracted from `internal/text` (grapheme + multiline
  helpers, dep `rivo/uniseg` only) — the widget layer no longer reaches into
  the shaping stack.
- **A5–A7** `internal/{hit,text,render,gpu,wgpu,vellum}` promoted to
  exported packages at their future homes.
- **A8** `scripts/check_boundaries.sh` enforces the DAG in CI.

## Phase B — extraction steps

History is preserved with [git filter-repo](https://github.com/newren/git-filter-repo)
(fresh clone per target repo, `--path` selectors, `--path-rename` where the
extracted directory becomes the module root). `git subtree split` is not
sufficient: several targets take multiple paths, and platform/wgpu need a
root rename.

Local development across repos uses a single workspace directory:

```sh
mkdir lux-dev && cd lux-dev
git clone git@github.com:timzifer/wgpu
git clone git@github.com:timzifer/lux-core
git clone git@github.com:timzifer/lux-render
git clone git@github.com:timzifer/lux-platform
git clone git@github.com:timzifer/lux
git clone git@github.com:timzifer/lux-app
git clone git@github.com:timzifer/lux-tools
go work init ./wgpu ./lux-core ./lux-render ./lux-platform ./lux ./lux-app ./lux-tools
```

`go.work` masks version requirements locally; **before tagging any release,
verify with `GOWORK=off go build ./... && GOWORK=off go test -tags nogui ./...`**
so hidden workspace-only dependencies cannot slip through.

### B1 — wgpu fork

```sh
git clone --no-local lux wgpu-extract && cd wgpu-extract
git filter-repo --path vendor_gogpu_wgpu --path-rename vendor_gogpu_wgpu/:
# rewrite the module path (go.mod + all self-imports):
grep -rl 'github.com/gogpu/wgpu' --include='*.go' --include='go.mod' . \
  | xargs sed -i 's|github.com/gogpu/wgpu|github.com/timzifer/wgpu|g'
git push git@github.com:timzifer/wgpu main
git tag v0.21.4 && git push --tags   # continuity with upstream v0.21.3
```

In the monorepo: remove the `replace` directive and `vendor_gogpu_wgpu/`,
rewrite the `github.com/gogpu/wgpu` imports (only `wgpu/`, `gpu/` and
`platform/cocoa` touch it) to `github.com/timzifer/wgpu`, then
`go mod tidy`.

### B2 — lux-core

```sh
git clone --no-local lux core-extract && cd core-extract
git filter-repo \
  --path draw --path input --path anim --path a11y --path interaction \
  --path validation --path fonts --path image --path theme \
  --path textseg --path hit --path LICENSE --path NOTICE
go mod init github.com/timzifer/lux-core && go mod tidy
# rewrite self-imports:
grep -rl 'github.com/timzifer/lux/' --include='*.go' . \
  | xargs sed -i 's|github.com/timzifer/lux/|github.com/timzifer/lux-core/|g'
git push git@github.com:timzifer/lux-core main && git tag v0.1.0
```

In the monorepo: delete those directories, rewrite imports
(`github.com/timzifer/lux/draw` → `github.com/timzifer/lux-core/draw`, etc.),
add `require github.com/timzifer/lux-core v0.1.0`.

### B3 — lux-render

Same procedure with `--path text --path render --path gpu --path wgpu`.
Carries the external deps `go-gl/gl`, `go-text/typesetting`, `pierrec/msdf`,
`gogpu/gputypes` and `github.com/timzifer/wgpu`.

### B4 — lux-platform

```sh
git filter-repo --path platform --path-rename platform/:
```

Root package `platform`, backends as subpackages (import paths become
`github.com/timzifer/lux-platform` and `github.com/timzifer/lux-platform/glfw`
etc.). Carries `go-gl/glfw`, `godbus/dbus`, `zzl/go-win32api`, `go-webgpu/goffi`.

### B5 + B6 — lux-app and lux-tools (one coordinated step)

These two must leave together: if `app` were extracted first, the monorepo's
`examples` (→ lux-app → lux/ui) would create a module require-cycle back into
the not-yet-cleaned monorepo.

- lux-app: `--path app --path dialog --path apptest --path internal/loop --path vellum`
- lux-tools: `--path examples --path cmd --path web/wasm`
- one monorepo cleanup commit removes both sets.

Tag lux-app `v0.1.0`; lux-tools stays untagged (leaf, applications only).

### B7 — lux (widgets) final state

This repo then contains only `ui/`, `richtext/`, `web/css`, `web/dom`,
`web` (acid harness), `uitest/`, `docs/` (incl. the RFCs — they document the
ui contract). Tag `v0.1.0`. The README gains the repo map above.
Port `scripts/check_boundaries.sh` to each repo as a plain
"module imports only its allowed modules" check (`go mod graph`).

## Versioning & release trains

- All modules `v0.x`; bump bottom-up (core minor bump → cascade PRs upward).
- `draw` is the universal vocabulary (`draw.Scene` is the render wire
  format). Keep changes additive; a breaking `draw` change is a coordinated
  release across all repos.
- Renovate/dependabot on each downstream repo keeps the requires fresh.

## CI per repo

| Repo | CI |
|---|---|
| wgpu | fork's own CI (kept from upstream) |
| lux-core | `go vet` + `go test ./...` (pure Go, no tags needed) |
| lux-render | `go vet` + `go test -tags nogui ./...`; optional real-GPU job |
| lux-platform | build-only cross-compile matrix: `GOOS=linux/windows/darwin/js` × tags (`x11`, `wayland`, `drm`, `nogui`) |
| lux (widgets) | `go test -tags nogui ./...` incl. golden + acid suites |
| lux-app | `go test -tags nogui ./...` (apptest runs headless) |
| lux-tools | build all examples, native + `GOOS=js GOARCH=wasm`, build lux-inspector |

## Risks

- **Golden files** (uitest) depend on `fonts` + `theme` from lux-core: pin
  the core version in the widgets go.mod; regenerate goldens only in
  deliberate "bump core" PRs.
- **Optional-interface probes** (`NativeHandle()`, `DRMfd()`, `A11yBridge()`,
  `SetAtlas(*text.GlyphAtlas)`, `WASMCanvas()`) cross repo boundaries via
  structural typing — the compiler will not catch signature drift. When
  touching one, grep all repos; consider promoting each probe to a named
  exported interface in the provider repo with `var _` assertions in the
  implementations.
- **Build-tag matrix** (`nogui`, backend tags) now spans four repos; the
  per-repo cross-compile CI is the guard.
