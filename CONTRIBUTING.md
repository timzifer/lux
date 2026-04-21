# Contributing to lux

Thanks for your interest in lux! This document covers the basics of
building, testing, and opening pull requests. Larger design changes
should start with an RFC in [`docs/rfc/`](docs/rfc/).

## Requirements

- **Go 1.25+**
- Native build toolchain appropriate to your platform backend
  (GLFW / Win32 / Cocoa / X11 / Wayland / DRM/KMS). Most CI jobs run
  with `CGO_ENABLED=0` and the `nogui` build tag, which matches the
  minimal path.

## Building and testing

The same commands used in CI are:

```sh
# Build and vet (matches the CI build tag)
go build -tags nogui ./...
go vet   -tags nogui ./...

# Unit tests (excluding vendored wgpu fork and UI golden tests)
go test -tags nogui -count=1 -timeout=10m \
  $(go list -tags nogui ./... | grep -v /vendor_gogpu_wgpu/ | grep -v /uitest)

# UI golden-file tests
go test -tags nogui -count=1 -timeout=10m ./uitest/...
```

Please run the relevant subset locally before opening a pull request.

## Vendored `vendor_gogpu_wgpu/`

lux ships a vendored fork of [gogpu/wgpu](https://github.com/andreykaipov/gogpu)
in `vendor_gogpu_wgpu/` because a Metal-backend bindgroup fix (see
[`docs/internal/gogpu-metal-bindgroup-fix.md`](docs/internal/gogpu-metal-bindgroup-fix.md))
is not yet upstream. Changes in that directory should ideally be
upstreamed; patches that only exist locally need to be documented in
the same fix note.

The vendored fork carries its own MIT license; see
[`vendor_gogpu_wgpu/LICENSE`](vendor_gogpu_wgpu/LICENSE) and the
root [`NOTICE`](NOTICE) for attribution.

## Pull requests

- Keep PRs focused — one logical change per PR.
- Prefix the subject with a Conventional-Commits-style type
  (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`) when it
  helps; it is not strictly enforced.
- Include tests for new behavior and ensure CI is green.
- Update affected documentation (`docs/`, `README.md`, inline
  package comments) in the same PR.
- Use the PR template checklist — it exists to keep reviewers fast,
  not to trip you up.

### Sign-offs (DCO)

Commits should be signed off to assert that you have the right to
contribute the change under the project's Apache 2.0 license:

```sh
git commit -s -m "feat: ..."
```

This adds a `Signed-off-by: Name <email>` trailer. See
<https://developercertificate.org/> for the certificate text.

## Reporting bugs and requesting features

Use the GitHub issue templates. For security vulnerabilities see
[`SECURITY.md`](SECURITY.md) — please do **not** open a public issue.

## Questions

Open a GitHub Discussion (once enabled) or a question-type issue.
