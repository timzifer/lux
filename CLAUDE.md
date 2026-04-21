# Repo conventions for Claude Code

## Git commit identity

All commits made in this repo must be authored as:

- Name: `timzifer`
- Email: `6703801+timzifer@users.noreply.github.com`

Rationale: the CLA Assistant workflow (`.github/workflows/cla.yml`)
allowlists the GitHub account `timzifer`. Commits attributed to other
identities (e.g. `Claude <noreply@anthropic.com>`) are flagged as unsigned
by the CLA bot, and because those accounts cannot themselves post the
`I have read the CLA Document and I hereby sign the CLA` comment, the
signature can never be recorded in `lux-cla-signatures/signatures/v1/cla.json`
and the PR stays blocked.

The workflow additionally allowlists `claude` and `*[bot]` as a safety
net, but please still commit under the `timzifer` identity so the
signatures file stays accurate for real contributors.

Use an inline override on every commit so it works regardless of the
session's default `user.name` / `user.email`:

```
git -c user.name=timzifer \
    -c user.email=6703801+timzifer@users.noreply.github.com \
    commit -m "..."
```

Do not amend existing commits from earlier sessions to change their author —
only apply this to new commits you create.
