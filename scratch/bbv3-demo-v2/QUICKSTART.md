# bbv3-demo-v2 — quickstart

Short command sequence. See `README.md` for the full story and rationale.

## Setup (once)

Runs against the locally-built provider via `dev_overrides` (built from current source, exercising the
v0.24.0 line). The global `~/.terraformrc` already dev-overrides `meshcloud/meshstack` to the repo root,
so `task build` is all the wiring needed — no `tofu init` for the meshstack provider.

```bash
cd ../..                                                # terraform-provider-meshstack repo root
task build                                              # build the dev provider binary from current source -> repo root
set -a && source .env && set +a                         # admin creds + MESHSTACK_ENDPOINT (http://localhost:8080)
cd scratch/bbv3-demo-v2
```

Prereqs: local meshStack up with the mux + tf + manual runner fan-out (see `local-dev-stack` /
`meshstack-services` skills). The BBD uses the terraform implementation, so the `tf-block-runner` must
be polling.

## Run

```bash
# 0 — bootstrap (admin): workspaces + scoped keys; export outputs to TF_VAR_*
cd 00_bootstrap && tofu init && tofu apply
for v in operator_client_id operator_client_secret appteam_client_id appteam_client_secret \
  platform_workspace appteam_workspace suffix; do export TF_VAR_$v=$(tofu output -raw $v); done
cd ..

# 1-2 — operator authors + tests the BBD (secret decrypts)
cd operator
TF_VAR_bbd_phase=draft-good tofu apply

# 3 — break it (same block re-runs in place -> FAILED; apply ERRORS with the run log)
TF_VAR_bbd_phase=draft-broken tofu apply        # expected error: "intentionally broken BBD version"

# 4 — release broken v1
TF_VAR_bbd_phase=v1-released tofu apply          # bbd_state -> RELEASED
cd ..

# 5 — app team consumes v1; run logs gated (apply SUCCEEDS with block eagerly PENDING, not tainted).
#     wait_for_completion stays FALSE on purpose: waiting would error the CREATE and TAINT the block,
#     forcing destroy+recreate at step 9 instead of an in-place reconcile (see appteam/main.tf).
cd appteam && TF_VAR_pin=v1 tofu apply           # succeeds immediately; status PENDING (backend eager-sets it)
# the run then fails on the broken ref; poll until the status settles off PENDING (~15-30s):
until tofu refresh >/dev/null 2>&1 && [ "$(tofu output -raw app_block_status)" != PENDING ]; do sleep 5; done
tofu output app_block_status                      # FAILED (apply succeeded, block NOT tainted)
tofu output app_block_latest_run_uuid            # an opaque run uuid — exposed to the app team, but the run's LOGS stay gated (run_transparency=false)
cd ..

# 6 — operator fixes + releases v2 (apply v2-released as its OWN step so feature is settled for step 7)
cd operator
TF_VAR_bbd_phase=v2-draft    tofu apply
TF_VAR_bbd_phase=v2-released tofu apply

# 7-8 — operator adopts + upgrades app-team blocks cross-workspace
TF_VAR_manage_appteam=true TF_VAR_bbd_phase=v2-released tofu apply
#   if "Invalid for_each argument": settle feature first, then converge:
#   ... tofu apply -target=meshstack_building_block_definition.feature ; ... tofu apply
cd ..

# 9 — app team reconciles the external upgrade
cd appteam
tofu plan                                        # drift: v1 -> v2
TF_VAR_pin=v2 tofu apply                          # reconcile (clean in-place)
cd ..
```

## Teardown

# The `managed` resource has `lifecycle { destroy = false }`, so the operator's destroy FORGETS the
# adopted app-team blocks (never deletes them). App team destroys its own block first (the operator
# then deletes the BBD it was made from).
```bash
cd appteam && tofu destroy
cd ../operator && TF_VAR_bbd_phase=v2-released tofu destroy
cd ../00_bootstrap && tofu destroy
```

## Notes

- Steps 3 errors by design (operator's broken re-run); the resource stays in state — read the output and continue.
- Keep each persona in its own folder; re-export the bootstrap `TF_VAR_*` if you open a fresh shell.
- `operator/` is split into `bbd.tf` (BBD + test block) and `managing_bbs.tf` (cross-workspace adopt/upgrade).
