# Security Policy

## Supported versions

lux is pre-1.0. Security fixes are applied to the `main` branch only.
Once tagged releases begin, this section will document the support
window.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security problems.**

Use GitHub's private vulnerability reporting for this repository:

> Security → Report a vulnerability
> <https://github.com/timzifer/lux/security/advisories/new>

If GitHub Security Advisories are not available to you, open a minimal
public issue asking for a private contact channel — do **not** include
details of the vulnerability in that issue.

Please include, as available:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected versions / commits,
- any suggested mitigation.

## What to expect

- Acknowledgement within **5 business days**.
- A triage decision (accepted / needs info / out of scope) within
  **14 business days**.
- Coordinated disclosure: we will agree on a public disclosure date
  with you before publishing an advisory. Credit will be given unless
  you prefer to remain anonymous.

## Scope

In scope:

- Source code in this repository (excluding `vendor_gogpu_wgpu/`,
  which should be reported to its upstream project when the issue is
  not specific to lux's fork).
- Documented build and release artifacts.

Out of scope:

- Vulnerabilities in third-party dependencies that have already been
  disclosed upstream — please reference the upstream advisory if you
  believe lux needs to pin or patch.
- Findings that require an attacker already having full control over
  the user's machine.
