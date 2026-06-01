# `tf-building-block` — bare git repo (test/demo fixture)

This directory **is a bare git repository** (note `HEAD`, `objects/`, `refs/` — no working tree, no
nested `.git`), so it is tracked in the provider repo as ordinary files, not a submodule. It holds a
no-op OpenTofu building-block module that the real `tf-block-runner` clones and runs **offline** (the
only network access is the OpenTofu binary download).

This `README.md` lives next to the git objects and is **not** committed inside the repo — a clone
yields only `main.tf`.

## Branches

| Branch | Content | Used by |
| --- | --- | --- |
| `main` | Single root commit. No-op module: declares the sensitive inputs (`api_key`, `script`, `static_secret`), runs a built-in `terraform_data` resource, and echoes the decrypted `api_key` back as the non-sensitive `api_key_echo` output (proves end-to-end sensitive-input decryption). | BB v3 acceptance test (clones `HEAD`); `bbv3-demo` / `bbv3-demo-v2` working BBD. |
| `broken` | `main` + one commit adding a `terraform_data` with a `precondition { condition = false }` that fails `apply` with a clear message. | `bbv3-demo-v2`, to release a deliberately failing BBD version (`ref_name = "broken"`); the fix is to point back at `main`. |

## How it is served

- **Acceptance test** serves it over git **smart-HTTP** (`git-http-backend` behind a Go `net/http/cgi`
  server, see `git_http_server_test.go`) and hands the runner an `http://127.0.0.1:<port>/…` clone
  URL — in CI the runner is a separate container, so `file://` would not be visible to it.
- **Local demos** (`scratch/bbv3-demo*`) run the runner via `go run` on the host, so they clone via
  `file://${abspath(...)}`.

Either way the runner clones a packfile, so the on-disk object encoding does not matter.

## Editing the module

A bare repo can't be edited in place — clone, edit, push back, then **garbage-collect and commit**:

```bash
src=internal/provider/testdata/tf-building-block            # this dir (run from repo root)
wd=$(mktemp -d); git clone "$src" "$wd/wd" && cd "$wd/wd"

# edit on main (keep it ONE root commit):  ...edit main.tf...  &&  git commit -a --amend --no-edit
# or edit the broken branch:               git checkout broken && ...edit...  && git commit -am '...'
git push --force origin <branch>                            # writes back into the bare repo

cd - && git -C "$src" gc --aggressive --prune=now           # repack into a single packfile
git add "$src" && git commit -m "update tf-building-block fixture"
```

**Always `git gc --prune=now` after pushing.** A push leaves loose objects (and possibly orphaned old
objects) under `objects/`; gc repacks them into one packfile + drops unreachable objects, keeping the
checked-in fixture small and deterministic. Then commit the changed `objects/` and `refs/`.

Keep `main` to a **single root commit** (the acc test clones `HEAD` with no `ref_name`, so whatever
`main` points at is what runs).
