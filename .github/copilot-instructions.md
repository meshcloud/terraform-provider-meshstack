# Copilot Instructions for meshStack Terraform Provider

Development conventions for this repo live in **[`AGENTS.md`](../AGENTS.md)** at the repository
root — the single source of truth for both AI agents and humans. Read it first.

Deeper, on-demand procedures live as skills under `.agents/skills/` (`.claude/skills` is a symlink to it):
- **`new-resource-datasource`** — add a resource/data source + its TestAcc test, with code exemplars.
- **`github-ci`** — GitHub Actions workflow conventions and action SHA-pinning.
- **`modern-go`** — Go 1.26 `new(expression)` and the codebase's generics.
- **`changelog-management`** — pick the next version and maintain `CHANGELOG.md`.
- **`meshstack-services`** — bring up a clean local meshStack backend (the legacy `meshfed-release` repo).
- **`acceptance-testing`** — run and debug the acceptance test suite.
