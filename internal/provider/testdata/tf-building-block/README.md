# `tf-building-block` — bare git repo (acc-test fixture)

This directory **is a bare git repository** (note `HEAD`, `objects/`, `refs/` — there is no working
tree and no nested `.git`, so it is tracked in the provider repo as ordinary files, not a submodule).

It holds a **single root commit** with a no-op OpenTofu building-block module (`main.tf`). The BB v3
terraform acceptance test clones it over a `file://` URL so the real `tf-block-runner` can `git clone`
and run OpenTofu **offline** (no GitHub, no network beyond the OpenTofu binary download) against a
module aligned with the terraform BBD in
`examples/resources/meshstack_building_block/test-support_04_sensitive_user_input_bbd.tf`.

This `README.md` is a plain file living next to the git objects — it is **not** committed inside the
repo, so it never appears in a clone (a clone yields only `main.tf`).

## Editing the module

You cannot edit a bare repo in place. Clone it, change it, and force-push a **single root commit** back:

```bash
src=internal/provider/testdata/tf-building-block          # this directory (run from repo root)
wd=$(mktemp -d)
git clone "$src" "$wd/wd" && cd "$wd/wd"

# ...edit main.tf...

# keep exactly one commit: amend the existing one
git commit -a --amend --no-edit
git push --force                                          # writes back into the bare repo

# OR rewrite history into a fresh single root commit:
#   git checkout --orphan fresh && git add -A \
#     && git commit -m "BB v3 acc test: no-op terraform building-block module" \
#     && git branch -M main && git push --force origin main
```

Then commit the updated bare repo (the changed files under `objects/` and `refs/`) in the provider repo.

Keep it **bare** and keep it to **one commit** — the test clones `HEAD` (the `main` branch) with no
`ref_name`, so whatever `main` points at is what runs.
