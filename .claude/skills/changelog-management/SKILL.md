---
name: changelog-management
description: Maintain CHANGELOG.md and pick the next version for this provider. Use when adding a changelog entry for a PR, deciding the next version number, or preparing main to be tagged/released. Derives the anticipated version from commits since the last git tag. Key rule — no "Unreleased" section; the top section always carries the concrete anticipated next version, and main is always ready to tag.
---

# Changelog & versioning

`CHANGELOG.md` drives releases (tags are cut from `main` via goreleaser). The governing
principle: **`main` is always ready to be tagged.** There is **no "Unreleased" section** — the
top section of `CHANGELOG.md` always names the concrete *anticipated next version*, and every
PR keeps it accurate.

## Versioning policy

This is a WIP `0.x` provider, so the major stays `0`.

- **Default: bump the PATCH** (`v0.21.0` → `v0.21.1`) — even for breaking changes. Most releases
  are patch bumps.
- **Bump the MINOR only occasionally**, when a *major update* lands (a large/significant change,
  not merely a breaking rename). Minor bumps reset patch to 0 (`v0.20.13` → `v0.21.0`).
- Never leave the next version unnamed.

## How the top section works

The latest git tag is the last *released* version. The top `CHANGELOG.md` section is the
*pending* release sitting on `main`:

- If the top section's version is **already greater than the latest tag**, it is the pending
  release — **add your entry to it**. (Today: latest tag `v0.21.0`, top section `v0.21.1` →
  pending; new entries go under `v0.21.1`.)
- If the top section's version **equals the latest tag** (no pending section yet), **create a new
  top section** with the bumped version (patch by default).
- If your change escalates the bump (a patch-level pending section, e.g. `v0.21.1`, now needs a
  minor because a major update landed), **rename the pending section** (`v0.21.1` → `v0.22.0`)
  rather than adding a second pending section.

A PR adding a feature therefore does **not** create an "Unreleased" block — it edits the named
pending section (or creates the next named one).

## Deriving the next version

```bash
git describe --tags --abbrev=0          # latest released tag, e.g. v0.21.0
LATEST=$(git describe --tags --abbrev=0)
git log "$LATEST"..HEAD --oneline       # commits since the tag
```

Classify the commits to choose categories and the bump:

| Commit type | Changelog category | Bump signal |
|---|---|---|
| `feat:` | FEATURES | patch (minor only if it's a major update) |
| `fix:` | FIXES | patch |
| breaking (`feat!`/`fix!`/`BREAKING CHANGE`) | BREAKING CHANGES | still patch by default |
| `docs:` / `chore:` / `refactor:` / `test:` / `ci:` | none — not user-facing | none |

Docs/chore/refactor/test/CI commits do **not** get changelog entries (e.g. the skills/docs
commits in this repo's history have none).

## Entry format

Match the existing style at the top of the file. Each version is its own section
(recent entries use a `# vX.Y.Z` heading) with these optional category blocks, in order:

```
# v0.21.1

Requires meshStack 2026.23.0 or later (previously 2026.22.0).   ← only when a newer backend is needed

BREAKING CHANGES:
- `meshstack_<resource>`: what changed and the migration the user must do.

FEATURES:
- `meshstack_<resource>`: what was added.

FIXES:
- `meshstack_<resource>`: what was fixed (and the symptom it resolves).
```

- Lead each bullet with the affected resource/data source in backticks when applicable.
- Describe user impact and any required migration, not the internal diff.
- Add the `Requires meshStack X.Y.Z or later` line when the change needs a newer meshStack
  version; note the previous requirement in parentheses.
- Omit empty category blocks.

## Checklist for a PR

1. `git log $(git describe --tags --abbrev=0)..HEAD --oneline` — see what's already pending.
2. Find or create the top pending section (version > latest tag; bump per policy — usually patch).
3. Add your bullet(s) under the right category.
4. Confirm no "Unreleased" heading exists and the top version number is the one you'd tag now.

See `AGENTS.md` → Code review (CHANGELOG entries are required for user-facing changes).
