# BBv3 demo v2 — cross-workspace release, break, fix & operator-driven upgrade

A runnable, step-by-step demo of the **`meshstack_building_block`** (v3) resource and the
`meshstack_building_block_definitions` / `meshstack_building_blocks` data sources, told as one story
across **three personas, three independent states**:

1. A **platform operator** authors a building block definition (BBD), tests it in its own workspace
   (with a real **sensitive input** decrypted end-to-end), then accidentally **breaks** it and
   **releases the broken version** anyway.
2. An **app team** in another workspace consumes the released BBD by name, hits the broken run, and —
   because the BBD has **`run_transparency = false`** and the app team holds only an unprivileged,
   workspace-scoped key — **cannot see the run logs**.
3. The operator **fixes** the BBD, releases a **v2**, then reaches **cross-workspace** with a
   `MANAGED_…` key to **adopt and upgrade** every app-team block made from that BBD — without owning or
   destroying them. The app team then sees the version move as an external change and reconciles its pin.

For the bare command sequence see [`QUICKSTART.md`](./QUICKSTART.md); this file is the story and the why.

## Personas, workspaces & keys

Three states, three providers. Bootstrap (admin) mints both scoped keys; the operator and app team then
act **only** through their own least-privilege key — that is the whole point.

| Persona | State | Key authorities | What it may do |
| --- | --- | --- | --- |
| **Admin** | `00_bootstrap/` | default `.env` admin creds | create the two workspaces + both scoped keys |
| **Platform operator** | `operator/` | `BUILDINGBLOCKDEFINITION_{SAVE,LIST,DELETE}` · `BUILDINGBLOCK_{SAVE,LIST,DELETE}` (its own test block) · `MANAGED_BUILDINGBLOCK_{SAVE,LIST}` + `MANAGED_BUILDINGBLOCKRUN_LIST` (reach into / read runs of other workspaces' blocks made from its BBD) · `ADM_BUILDINGBLOCKDEFINITION_SAVE` + `ADM_REVIEW_PUBLICATION` (demo shortcut, see below) | author/release/break/fix the BBD; adopt + upgrade app-team blocks cross-workspace |
| **App team** | `appteam/` | `BUILDINGBLOCK_{SAVE,LIST,DELETE}` (its own block) · `BUILDINGBLOCKDEFINITION_LIST` (find the BBD by name) | create/update its own block; cannot change the BBD version, cannot see run logs |

> **Why `MANAGED_…` reaches cross-workspace.** The operator key is owned by the **platform** workspace
> that owns the **definition**. `MANAGED_BUILDINGBLOCK_SAVE`/`_LIST` let it list and PUT blocks created
> from *its* definitions in any workspace; `MANAGED_BUILDINGBLOCKRUN_LIST` lets it read those blocks'
> run logs as the definition owner — which is what surfaces the failed run on the broken ref even though
> `run_transparency = false`. None of these is a delete authority — the operator never deletes the app
> team's block (see Teardown).
>
> **Demo shortcut.** The two `ADM_…` authorities are *not* least-privilege: a real operator would route a
> release through meshStack's publication-approval flow. They let the operator self-release draft versions
> (steps 4 / 6b) non-interactively. Drop both to see the realistic approval flow.

## Layout

```
bbv3-demo-v2/
├── 00_bootstrap/   admin: 2 workspaces + 2 scoped keys; outputs (creds, workspace names, suffix)
├── operator/       bbd.tf (BBD + operator test block) · managing_bbs.tf (cross-workspace adopt+upgrade)
└── appteam/        main.tf (the app team's consuming block)
```

Each folder has its **own default provider** for that persona (no aliases). State is **not shared** —
each folder has its own `terraform.tfstate`; personas pass data forward via `TF_VAR_*` sourced from the
`00_bootstrap` outputs. The terraform-implementation BBD clones the bundled bare fixture repo
`internal/provider/testdata/tf-building-block` over `file://`; it ships both a `main` (working, echoes the
decrypted `api_key`) and a `broken` (failing precondition) branch — nothing to set up.

## Steering variables

The configs progress through the story via a few variables rather than extra step folders.

**`operator/`** is driven by `bbd_phase` (the BBD lifecycle phase) — `bbd.tf`'s `local.phase` maps it to the terraform ref and draft flag:

| `TF_VAR_bbd_phase` | `ref_name` | `draft` | BBD state | Step |
| --- | --- | --- | --- | --- |
| `draft-good`   | `main`   | `true`  | working draft                       | 1–2 |
| `draft-broken` | `broken` | `true`  | broken draft (same version uuid)    | 3 |
| `v1-released`  | `broken` | `false` | v1 released, broken                 | 4 |
| `v2-draft`     | `main`   | `true`  | v2 draft, fixed (+ defaulted `size`)| 6a |
| `v2-released`  | `main`   | `false` | v2 released, fixed                  | 6b, 7, 8 |

The `v2-*` phases also add a **defaulted** operator input (`size`, `default_value = 16`) that v1 lacks, so
the phase is both a code fix and a definition-shape change. Plus `TF_VAR_manage_appteam` (bool, default
`false`) enables the cross-workspace adopt+upgrade (steps 7–8).

**`appteam/`** pins its block to a version via `TF_VAR_pin` ∈ {`v1`, `v2`} (`versions[0]` vs `versions[1]`,
sorted ascending).

> A released version is **immutable** — flipping `draft` back to `true` with changed content makes the
> provider create the *next* version as a fresh draft, never mutate the released one. That is why v2 is
> reached as `v2-draft` → `v2-released`.

## Walk-through

Run each persona from its own folder; export the bootstrap outputs to `TF_VAR_*` before the operator/app-team applies (see QUICKSTART for the exact commands).

**0 — Bootstrap (admin).** `cd 00_bootstrap && tofu init && tofu apply` → two workspaces + two scoped keys.
Export its outputs (`operator_*`/`appteam_*` creds, workspace names, `suffix`) into `TF_VAR_*`.

**1–2 — Operator authors & tests the BBD** (`bbd_phase=draft-good`). Creates a **`WORKSPACE_LEVEL`**
terraform BBD (`ref_name=main`, `draft=true`, `run_transparency=false`) with user inputs only — including a
**sensitive** `api_key`. The operator's test block in the platform workspace runs to `SUCCEEDED` and the
module echoes the decrypted secret (`status.outputs.api_key_echo` == plaintext; `all_inputs.api_key` is a
hash only) — end-to-end sensitive-input proof.

**3 — Operator breaks it** (`bbd_phase=draft-broken`). Only `ref_name` flips `main → broken`; the draft keeps
the **same version uuid**, so its `content_hash` changes and the test block (tracking `version_latest`)
**re-runs in place** — no replace. The run fails on the broken ref and, with `wait_for_completion = true`,
the **apply ERRORS and surfaces the failure log** (the operator can read it as BBD owner). The errored apply
does not block the next one.

**4 — Operator releases the broken v1** (`bbd_phase=v1-released`). v1 is now released and broken (`bbd_state → RELEASED`).

**5 — App team consumes v1; run fails; run logs gated** (`appteam/`, `pin=v1`). The `meshstack_building_block_definitions`
data source finds the BBD by display name (filtered in HCL — no name filter param); the block pins
`versions[0]`. v1 has only user inputs, so it **runs** and **fails** on the broken ref. `wait_for_completion = false`
is deliberate: waiting would error the *create* and **taint** the block, so step 9 would destroy+recreate
instead of reconciling in place. With `false` the apply **succeeds** immediately — the backend **eagerly sets
the block to `PENDING`** the moment its inputs make it runnable, so right after apply `app_block_status == PENDING`;
the run then fails on the broken ref a few seconds later and a `tofu refresh` moves it to `FAILED`. Once a run
exists `status.latest_run_uuid` is an **opaque run uuid** the app team can read, but the run's **logs stay gated**
by `run_transparency = false` — the app team sees *that* a run happened, not *what* it did (`tofu output
app_block_latest_run_uuid`).

**6 — Operator fixes & releases v2** (`bbd_phase=v2-draft` then `v2-released`, each its own apply). `ref_name`
returns to `main` and the BBD gains a **defaulted** operator input `size = 16`. Because it is defaulted, the
upgrade in steps 7–8 need not supply it. The operator test block goes green again on the fixed ref.

**7–8 — Operator adopts & upgrades app-team blocks cross-workspace** (`manage_appteam=true`, keep
`bbd_phase=v2-released`). `data.meshstack_building_blocks.managed` (scoped by `managed_by_definition_uuid`)
lists every block made from the definition; `local.managed` **excludes the operator's own test block** (it
lives in the platform workspace and is managed in `bbd.tf`). An `import` block keyed off `local.managed`
adopts each remaining block into operator state, and the `managed` resource bumps it to the released v2 with
`inputs = {}` — a pure version bump: the backend fills `size = 16` and the app team's user inputs (incl.
`api_key`) are preserved. `tofu output managed_building_blocks` lists the adopted blocks.

> **Settle the BBD first.** An `import` `for_each` must resolve at **plan** time, and `local.managed` comes
> from a data source OpenTofu only reads at plan when `feature` has **no** pending change. Apply the v2
> release (step 6b) in its own apply before enabling `manage_appteam`. If you still hit
> `Invalid for_each argument`, converge in two steps:
> `tofu apply -target=meshstack_building_block_definition.feature` then `tofu apply`.

**9 — App team reconciles the external change** (`appteam/`). `tofu plan` shows the version drifted v1→v2;
repinning `pin=v2` makes the plan clean and applies in place. A version change requested by the app team's
*own* key would be rejected (403) — only the operator/admin may change the BBD version.

## Teardown

The operator adopted (but does not own) the app team's blocks, so its state must come down without deleting
them. The `managed` resource carries `lifecycle { destroy = false }` (an OpenTofu-only customization —
"forget", never delete), so `tofu destroy` drops the adopted blocks from operator state and deletes only what
the operator owns (the BBD + test block). No `removed` block or `tofu state rm` needed. Order matters:

1. **App team** drops its block: `cd appteam && tofu destroy`.
2. **Operator** destroys its state (keep `bbd_phase=v2-released` — reverting would try to rewrite a released
   version): `cd operator && tofu destroy`. The adopted `managed[*]` blocks are **forgotten, not deleted**.
3. **Bootstrap** last: `cd 00_bootstrap && tofu destroy`.
