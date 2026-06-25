---
name: scratch-config-testing
description: Build and run a meshStack Terraform config in the git-ignored scratch/ dir against a local meshStack, using the dev-built provider binary via dev_overrides. Use when reproducing or debugging a provider bug or a failing acceptance test as a standalone, re-runnable config — by dumping a test's HCL or hand-writing one.
---

# scratch-config-testing

Turn an acceptance test's generated HCL (or hand-written HCL) into a **standalone,
re-runnable** Terraform config under git-ignored `scratch/`, run with the locally-built
provider against a local meshStack. Complements the **acceptance-testing** skill (brings up the
backend and runs the suite). The `testconfig` builders
make a dumped config self-contained: applied to an empty meshStack it creates its full
dependency chain (workspace → dependent resources).

## Prerequisites

1. Local meshStack up — see the **acceptance-testing** skill (Backend bring-up).
2. Provider env vars exported (the `http://localhost` guard applies):

   ```bash
   set -a && source .env && set +a   # MESHSTACK_ENDPOINT, MESHSTACK_API_KEY, MESHSTACK_API_SECRET
   ```

## One-time setup

Build the provider to the repo root, then point Terraform at it via a dev-override CLI config
kept separate from your global `~/.terraformrc`:

```bash
task build                                    # -> ./terraform-provider-meshstack
cat > "$HOME/.terraformrc.dev" <<EOF
provider_installation {
  dev_overrides {
    "meshcloud/meshstack" = "$PWD"            # dir containing the built binary (repo root)
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE=$HOME/.terraformrc.dev
```

With `dev_overrides`, Terraform prints a warning and **skips `terraform init`** — run
`terraform plan`/`apply` directly. Rebuild (`task build`) after any provider change.

## Get a config into scratch/

**From a test** — `MESHSTACK_SCRATCH_DUMP=1` makes `ApplyAndTest` dump each step's HCL and
return *without running the test* (no backend needed, fast, works even for configs that fail
to apply):

```bash
MESHSTACK_SCRATCH_DUMP=1 go test -run 'TestAccPaymentMethod$' ./internal/provider/ -v
```

Produces, under repo-root `scratch/`:

```text
scratch/TestAccPaymentMethod/step01/{main.tf,provider.tf}   # create
scratch/TestAccPaymentMethod/step02/{main.tf,provider.tf}   # update
```

`stepNN` is 1-based and matches the framework's step order; import-only steps (no `Config`)
are skipped. Subtests nest, e.g. `scratch/TestAccBuildingBlock/azure_devops/step01/`. Set the
env var to a path instead of `1` to dump elsewhere.

**Hand-written** — drop a `main.tf` into `scratch/repro/` and copy a `provider.tf` next to it
(same block the dump emits: `required_providers { meshstack = { source = "meshcloud/meshstack" } }`
plus an empty `provider "meshstack" {}`).

## Run it

```bash
cd scratch/TestAccPaymentMethod/step01
terraform plan        # no init needed with dev_overrides
terraform apply
terraform destroy      # clean up the meshObjects when done
```

Provider-side logs: `TF_LOG_PROVIDER=debug terraform apply`. To step through with a debugger,
build with `go build -gcflags="all=-N -l"` and attach delve to the running provider process.

## terraform-implementation BBDs (real `tf-block-runner`)

The standard local fan-out from the **`local-dev-stack`** skill already runs
the **`tf-block-runner`** behind the multiplexer (mux `:8300`), so a `meshstack_building_block_definition`
whose `implementation.terraform` clones a repo and runs OpenTofu — plus its consuming
`meshstack_building_block` — works in `scratch/` with **no runner swap**. (This is the same fan-out the
acceptance suite uses; nothing special is needed for `scratch/` play.)

The tf-block-runner downloads OpenTofu via tofudl, clones the BBD's `repository_url`, and for a module
that declares no backend injects the mesh http backend (`use_mesh_http_backend_fallback = true`). Watch
`/tmp/tf-runner.log`; the building block reaches `SUCCEEDED` with real tofu outputs in TF state.
Sensitive inputs (`sensitive = { argument = { secret_value = ... } }`) decrypt end to end out of the
box — the dev seed registers `building-blocks.pem` on the magic runner UUID and the tf-block-runner
ships the matching private key. A minimal BBD `implementation`:

```hcl
implementation = {
  terraform = {
    repository_url                 = "https://github.com/meshcloud/meshstack-hub.git"
    repository_path                = "modules/meshstack/noop/buildingblock"   # NoOp reference module: all input/output types, no real infra
    ref_name                       = "main"
    terraform_version              = "1.9.0"                                   # >1.5.5 → OpenTofu via tofudl
    use_mesh_http_backend_fallback = true
  }
}
```

One gotcha when hand-writing such a config:
- **`draft = false` versions are immutable** (`Updating a version_spec in non-draft state is not
  allowed`). Use `draft = true` to iterate — each apply updates the version in place and reruns
  the block; flip to `false` only to "release". A released version can't return to draft (destroy
  + recreate).

## Notes

- The built-in `TF_ACC_PERSIST_WORKING_DIR=1` only *preserves* a test's temp working dir for
  inspection — its provider is injected in-process via reattach, so that dir is **not**
  runnable standalone. This skill's output is, because of the `provider.tf` + dev_override.
- `scratch/` is git-ignored; delete freely with `rm -rf scratch/`.
