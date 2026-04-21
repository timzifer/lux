# Repo conventions for Claude Code

## Git commit identity

All commits made in this repo must be authored as:

- Name: `timzifer`
- Email: `6703801+timzifer@users.noreply.github.com`

Rationale: the CLA Assistant workflow (`.github/workflows/cla.yml`) only
allowlists the GitHub account `timzifer`. Commits attributed to any other
identity (e.g. `Claude <noreply@anthropic.com>`) are flagged as unsigned by
the CLA bot and block PR merges.

Use an inline override on every commit so it works regardless of the
session's default `user.name` / `user.email`:

```
git -c user.name=timzifer \
    -c user.email=6703801+timzifer@users.noreply.github.com \
    commit -m "..."
```

Do not amend existing commits from earlier sessions to change their author —
only apply this to new commits you create.
