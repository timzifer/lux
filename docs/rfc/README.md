# RFC Index

This directory contains the internal design documents (RFCs) for **lux**.

RFCs are written in German and describe the intended design, rationale, and implementation status
of each subsystem. They are the authoritative source for architecture decisions. The status
column below reflects the current implementation state.

| RFC | File | Status | Scope (English summary) |
|-----|------|--------|-------------------------|
| 001 | [RFC-001-lux.md](RFC-001-lux.md) | Partially integrated | Core architecture: Elm app loop, widget system, virtual tree reconciliation, rendering pipeline, platform abstraction, accessibility |
| 002 | [RFC-002-interaction-layout.md](RFC-002-interaction-layout.md) | Integrated | Input handling, focus management, animation system, Flexbox/Grid/Table layout engines, kinetic scrolling, overlay/effects |
| 003 | [RFC-003-lux-widget-catalogue.md](RFC-003-lux-widget-catalogue.md) | Integrated | Full widget catalogue, design token system, typography, i18n, RTL, built-in fonts |
| 004 | [RFC-004-hmi-touch.md](RFC-004-hmi-touch.md) | Integrated | Touch input, multi-touch gesture recognition, HMI/embedded targets |
| 004 | [RFC-004-lux-webview.md](RFC-004-lux-webview.md) | Partially integrated | WebView integration as an external surface (WebView2 on Windows) |
| 006 | [RFC-006-lux-surface-semantics.md](RFC-006-lux-surface-semantics.md) | Integrated | External surface API, zero-copy texture provider, accessibility semantics for surfaces |
| 007 | [RFC-007-lux-WGPU-rendering.md](RFC-007-lux-WGPU-rendering.md) | Partially integrated | GPU rendering via wgpu/gogpu (Vulkan/Metal/D3D12), geometry batcher, visual effects, OpenGL fallback |
| 008 | [RFC-008-lux-default-theme.md](RFC-008-lux-default-theme.md) | Integrated | Lux dark/light theme (design tokens, color system, typography, motion, elevation) |
| 010 | [RFC-010-lux-code-editor.md](RFC-010-lux-code-editor.md) | Planned | Code editor widget (syntax highlighting, virtual rendering, multi-cursor — not yet implemented) |
| 011 | [RFC-011-lux-vellum.md](RFC-011-lux-vellum.md) | Theoretical | Remote/network rendering protocol for distributed UI (not scheduled for implementation) |
| 998 | [RFC-998-lux-browser-engine.md](RFC-998-lux-browser-engine.md) | Theoretical | Custom browser engine integration (exploratory, not planned) |
| 999 | [RFC-999-lux-sim.md](RFC-999-lux-sim.md) | Integrated | Testing infrastructure: headless simulation, golden-file scene tests |

## Status Definitions

| Status | Meaning |
|--------|---------|
| **Integrated** | Design is fully implemented; RFC is the historical rationale |
| **Partially integrated** | Core design is implemented; some sections remain pending |
| **Planned** | Design is approved; implementation has not started yet |
| **Theoretical** | Exploratory document; not on the implementation roadmap |
