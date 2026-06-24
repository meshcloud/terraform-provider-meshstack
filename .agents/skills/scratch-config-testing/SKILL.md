---
name: scratch-config-testing
description: Build and run a meshStack Terraform config in the git-ignored scratch/ dir against a local meshStack, using the dev-built provider binary via dev_overrides. Use when reproducing or debugging a provider bug or a failing acceptance test as a standalone, re-runnable config — by dumping a test's HCL or hand-writing one.
---

# scratch-config-testing

Turn an acceptance test's generated HCL (or hand-written HCL) into a **standalone,
re-runnable** Terraform config under git-ignored `scratch/`, run with the locally-built
provider against a local meshStack. Complements the **acceptance-testing** skill (runs the
suite) and **meshstack-services** skill (brings up the backend). The `testconfig` builders
make a dumped config self-contained: applied to an empty meshStack it creates its full
dependency chain (workspace → dependent resources).

## Prerequisites

1. Local meshStack up — see the **meshstack-services** skill.
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

## Optional: terraform-implementation BBDs (real `tf-block-runner`)

> **Not needed for acceptance testing.** The acceptance suite (and the default local backend
> from the **meshstack-services** skill) registers the **manual** block runner — a no-op that
> echoes inputs as outputs and never executes terraform. Leave that as-is for `task testacc`.
> This section is only for *playing in `scratch/` with a `meshstack_building_block_definition`
> whose `implementation.terraform` actually clones a repo and runs OpenTofu*, plus its
> consuming `meshstack_building_block`.

The manual runner can't run terraform. To exercise a terraform-impl BBD end-to-end, swap it for
the **`tf-block-runner`** — a Go app in the `building-block-runner` repo (may sit at
`../building-block-runner/tf-block-runner`; it is *not* a gradle module, so `go run .`):

```bash
pkill -9 -f "BlockRunnerApplicationKt"          # stop the manual runner (same UUID would race)
cd ../building-block-runner/tf-block-runner
RUNNER_UUID=98520496-627d-43e6-82da-ce499179ff3f \
  RUNNER_API_CLIENT_ID=<local managed-runners client id> \
  RUNNER_API_CLIENT_SECRET=<local managed-runners secret> \
  nohup go run . > /tmp/tf-runner.log 2>&1 &     # restart the manual runner afterwards for acc tests
```

`RUNNER_UUID` (env) overrides `runner-config.yml` and **must** equal the
`SharedBuildingBlockRunnerUuid` (`internal/provider/building_block_runner.go`,
`98520496-…`) that BBD examples default to, or runs sit unclaimed. It authenticates with the local
managed-runners API key (`RUNNER_API_CLIENT_ID`/`RUNNER_API_CLIENT_SECRET` above — seeded dev values
are in the meshfed-release `local-dev-stack` skill, kept out of this public repo; the tf-block-runner
no longer ships a basic-auth default). The runner polls `localhost:8080`, downloads OpenTofu via
tofudl, clones the BBD's
`repository_url`, and for a module that declares no backend injects the mesh http backend
(`use_mesh_http_backend_fallback = true`). Watch `/tmp/tf-runner.log`; the building block
reaches `SUCCEEDED` with real tofu outputs in TF state. A minimal BBD `implementation`:

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

Two gotchas when hand-writing such a config:
- **Sensitive STATIC inputs fail to decrypt** (`file input decryption failed`): the local shared
  runner's registered public key doesn't match the `tf-block-runner`'s `runner-config.yml` key.
  For play, declare the input as plain STATIC (`argument = jsonencode(...)`) instead of
  `sensitive = { argument = { secret_value = ... } }`.
- **`draft = false` versions are immutable** (`Updating a version_spec in non-draft state is not
  allowed`). Use `draft = true` to iterate — each apply updates the version in place and reruns
  the block; flip to `false` only to "release". A released version can't return to draft (destroy
  + recreate).

## Notes

- The built-in `TF_ACC_PERSIST_WORKING_DIR=1` only *preserves* a test's temp working dir for
  inspection — its provider is injected in-process via reattach, so that dir is **not**
  runnable standalone. This skill's output is, because of the `provider.tf` + dev_override.
- `scratch/` is git-ignored; delete freely with `rm -rf scratch/`.
