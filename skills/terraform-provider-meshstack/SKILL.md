---
name: terraform-provider-meshstack
description: >
  Manage meshStack building block definitions and building blocks as code with the meshStack
  Terraform/OpenTofu provider. Covers login, looking up objects, creating and releasing a
  definition, adding, changing, upgrading and deleting a building block, waiting for runs and
  approvals, and importing what exists. Use when asked to write or change `meshstack_*` HCL, to
  order or update a building block through Terraform, to publish a definition version, or to bring
  meshStack objects under Terraform. In a clone of the provider repository, also use it to dump an
  acceptance test's HCL and to run a config against a dev build, to reproduce a provider bug or to
  prototype a demo.
---

# meshStack Terraform provider

This skill is for agents and people who **use** the provider. Its last section,
[Run a config against a dev build](#run-a-config-against-a-dev-build), is for contributors who
change the provider. It lives in the provider repository, so it changes together with the
resources.

New to meshStack? Start at the [meshStack agent context](https://docs.meshcloud.io/agent-context/index.md).
It explains what meshStack is, the concepts, and the tasks, and it links back here for the HCL.

The [registry documentation](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs)
describes every attribute. When this skill and the registry disagree, trust the registry. Checking
runs and reading their logs belongs to the
[`meshstack-cli` skill](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md).
To build a definition from a ready-made module, start with the
[`meshstack-hub` skill](https://raw.githubusercontent.com/meshcloud/meshstack-hub/main/skills/meshstack-hub/SKILL.md).

| Resource | Manages |
|---|---|
| [`meshstack_building_block_definition`](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs/resources/building_block_definition) | a definition and all its versions, in one resource |
| [`meshstack_building_block`](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs/resources/building_block) | one building block on a workspace or a tenant |

`meshstack_building_block_v2` and `meshstack_buildingblock` are deprecated. Write
`meshstack_building_block`, and move old ones as [Import existing objects](#import-existing-objects)
shows. The commands below use `tofu`; `terraform` takes the same arguments.

## Get a working config

This skill holds no full configurations, because copies here would fall behind the provider. Two
sources are tested against a real meshStack, so start from them and replace the names and values
with the user's:

- **The examples of the provider.** The acceptance tests apply them, and the registry shows them as
  "Example Usage". Each section below links the ones it needs. The links point to `main`. When the
  user runs an older provider version, replace `main` with its tag, such as `v0.27.0`.
- **The configurations of the acceptance tests**, when you work in a clone of this repository:

  ```sh
  MESHSTACK_SCRATCH_DUMP=1 go test -run 'TestAccBuildingBlock$' ./internal/provider/
  ```

  This writes the HCL of every step of that test to `scratch/<test>/stepNN/main.tf` and
  `provider.tf`, without running the test and without a meshStack. `stepNN` counts from 1 in the
  order of the test's steps and skips steps that only import. A subtest gets a folder of its own,
  such as `scratch/TestAccBuildingBlock/02_tenant/step01/`. Set the variable to a path instead
  of `1` to write there. Each step is complete: applied to an empty meshStack, it creates everything
  it needs, starting with the workspace.

An example sets many optional attributes to show them. Keep only what the user needs.

## Log in

Act as the user, through their own meshStack CLI browser login. meshStack then records the user as
the author of every change. The CLI skill shows
[how an agent starts that login](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#logging-in-as-an-agent).
The provider uses the stored login, renews it, and never opens a browser. When the login has
expired, the provider fails, and the user has to log in again.

Declare the provider with `source = "meshcloud/meshstack"` in `required_providers`. The provider
block takes the CLI profile and the workspace, as [provider.tf](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/provider/provider.tf) shows.

- A browser login needs `workspace`, or `MESHSTACK_WORKSPACE`. The provider ignores the default
  workspace of the profile, so the configuration acts in the same workspace for everyone who
  applies it. Use the identifier meshPanel shows, such as `devops-platform`.
- The login acts with the user's role in that workspace. Set it to the workspace the objects belong
  to: the platform team's workspace for a definition, the building block's workspace for a block.
- For several workspaces, write one provider block per workspace with an `alias`, and set
  `provider = meshstack.<alias>` on each resource, as
  [provider_workspaces.tf](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/provider/provider_workspaces.tf) shows.
- A credential in the environment wins over the profile. Unset `MESHSTACK_API_KEY`,
  `MESHSTACK_API_SECRET` and `MESHSTACK_API_TOKEN` to act as the user.

An agent that runs without a human, such as a CI job or a scheduled run, uses an API key instead.
Give the key only the [permissions](https://docs.meshcloud.io/api/authentication/api-permissions/)
the configuration needs. Leave the provider block empty and set `MESHSTACK_ENDPOINT`,
`MESHSTACK_API_KEY` and `MESHSTACK_API_SECRET` from the secret store of the CI system. Never set
`apisecret` in the provider block: Terraform then writes the secret to state. An API key needs no
workspace.

## Find existing objects

Read what exists with data sources and pass their `ref` and version outputs on, rather than copying
uuids into the configuration. Examples:
[workspace](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/data-sources/meshstack_workspace/data-source.tf),
[definition](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/data-sources/meshstack_building_block_definition/data-source.tf),
[tenants](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/data-sources/meshstack_tenants/data-source.tf),
[payment method](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/data-sources/meshstack_payment_method/data-source.tf) and
[building blocks](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/data-sources/meshstack_building_blocks/data-source.tf).

- The `ref` of a workspace is the `target_ref` of a workspace-level building block. For a
  tenant-level block, filter `meshstack_tenants` by `platform = "<platform>.<location>"` and take
  `one(data.meshstack_tenants.<name>.tenants).ref`.
- The definition data source fails unless exactly one definition matches. Add
  `metadata.owned_by_workspace` when two workspaces offer a definition of the same name.
- `version_latest_release` of a definition is its newest released version, `version_latest` the
  newest including a draft, and `versions["v2"]` version 2.
- `meshstack_building_blocks` filters by `workspace_identifier`, `name`, `definition_uuid`,
  `health_status`, `action` and more: see its
  [registry page](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs/data-sources/building_blocks).

The provider does not show which inputs a definition version declares. Read them with the CLI, as
[Ordering a building block](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#ordering-a-building-block)
describes.

## Create a building block definition

A definition belongs to the platform team's workspace. It starts as a draft, with
`version_spec.draft = true`. Start from the example of its implementation:
[terraform](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block_definition/resource_01_terraform.tf),
[github_workflows](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block_definition/resource_02_github_workflows.tf),
[manual](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block_definition/resource_03_manual.tf),
[azure_devops_pipeline](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block_definition/resource_04_azure_devops_pipeline.tf) or
[gitlab_pipeline](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block_definition/resource_05_gitlab_pipeline.tf).

- `default_value` and `argument` are `jsonencode`d to match `type`, also for a string.
- A `TENANT_LEVEL` definition also needs `spec.supported_platforms`. `target_type` cannot change:
  a changed `target_type` replaces the definition.
- Approval gates (`spec.approval_policies`) and drift detection need the `terraform`
  implementation. See the
  [registry page](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs/resources/building_block_definition)
  for every input and output attribute.
- The example writes secrets such as `ssh_private_key.secret_value` and a sensitive `argument` as
  literals. Feed them from an ephemeral variable instead, as for a sensitive input of a building
  block.
- Only the owning workspace can order a draft. Test the draft there with a building block whose
  `building_block_definition_version_ref` is the definition's `version_latest`.
  The `content_hash` in `version_latest` changes with every edit of the draft, so the next apply
  runs the test block again.

## Release a version

Set `version_spec.draft = false`, then plan and apply. The latest version is released, and
`version_latest_release` points to it.

- When the release needs the approval of a meshStack admin, the apply warns
  `Building Block Definition release needs admin approval`. Tell the user. After an admin approved
  the release in meshPanel, plan again: the warning is gone.
- A released version cannot change. To change anything in `version_spec`, set `draft = true` in
  the same edit. The apply creates the next version number as a draft. Test it, then set
  `draft = false` again to release it. A change to `version_spec` with `draft = false` shows in the
  plan, but the apply fails with `Updating a version_spec in non-draft (released) state is not allowed`
  before it changes anything.
- `spec` is not versioned. A changed display name, description or approval policy applies to the
  definition at once, also while its latest version is released.
- Existing building blocks stay on their version until someone upgrades them, as
  [Change the inputs of a building block](#change-the-inputs-of-a-building-block) shows.

## Add a building block

Start from the example for a
[workspace](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block/resource_01_workspace.tf)
or a
[tenant](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block/resource_02_tenant.tf).
Keep its postcondition on `self.status.succeeded`: without it, a failed run is only a warning, and
the apply looks successful.

- Order a released version with the definition's `version_latest_release`. The examples use
  `version_latest`, which is a draft while the definition has one.
- The examples set `timeouts` of 2 minutes for the tests. Leave `timeouts` out for the default of
  30 minutes, or raise them for slow runs.
- Set the inputs the version declares as `USER_INPUT` or `PAYMENT_METHOD`. meshStack and the
  platform team fill in the others. `inputs` is required, so write `inputs = {}` when you set none.
- Every `value` is `jsonencode`d, also a string. A `PAYMENT_METHOD` value is the `jsonencode`d
  `ref` of a `meshstack_payment_method` data source.
- `version_latest_release` is null until the definition has a released version.

A sensitive input takes `sensitive = { secret_value = var.<name>, secret_version = "1" }` instead of
`value`. The tenant example writes a literal there; declare the variable with `sensitive = true` and
`ephemeral = true` instead, so the value is neither in the code nor in state. Change
`secret_version` to send a new value. Set `TF_VAR_<name>` for the plan and again for the apply,
because a saved plan does not keep an ephemeral value. This needs Terraform or OpenTofu 1.11 or
later.

## Change the inputs of a building block

Change the `value` in `spec.inputs`, then plan. The plan shows an in-place update (`~`), and the
apply starts a run and waits for it.

- A changed `display_name` renames the block without a run.
- An input you drop from `spec.inputs` keeps its value in meshStack. Set the new value instead.
- meshStack rejects a change to an input that the definition does not mark
  `updateable_by_consumer`, unless the login is the platform team's or an admin's.
- To upgrade, point `building_block_definition_version_ref` at the definition's
  `version_latest_release`. The block is upgraded in place, and a run follows. meshStack accepts
  only the latest released version, and only while the block is not running or waiting. Only a
  login of the platform team that owns the definition, or of an admin, may change the version.
- A changed `target_ref`, or a changed `parent_building_block_refs` without an upgrade, replaces
  the block (`-/+`): its destroy run removes what it created. Ask the user before you apply that.
- A block whose last run failed shows an in-place update on every plan, also without a change. That
  apply runs the block again.

## Delete a building block

Remove the resource block from the configuration, then plan. The plan shows the block as destroyed
(`-`). Ask the user before you apply it.

The apply starts a destroy run in meshStack, which removes what the block created, and waits until
the block is gone, up to `timeouts.delete`. When that time runs out, the apply fails and keeps the
block in state, so the next apply waits again. meshStack does not delete a definition while
building blocks still use it.

To stop managing a block without deleting it, replace its resource block with:

```hcl
removed {
  from = meshstack_building_block.repo

  lifecycle {
    destroy = false
  }
}
```

`purge_on_delete = true` removes a block without a destroy run and leaves its cloud resources
behind without management. It needs an admin. Use it only when the user asks for it, for a deletion
that is stuck. Apply it before you remove the resource block: the delete reads it from state.

## Wait for runs and approvals

The apply of a create or an update waits for the run to end, up to `timeouts` (30 minutes by
default). Raise them for slow runs, for example `timeouts = { create = "60m", update = "60m" }`.
The apply reports a run that does not succeed as a warning:

| The apply reports | What it means | What to do |
|---|---|---|
| `Building block run did not succeed` | the run failed or was aborted; the next warning names the first failed step | read the run, as [Why a run failed](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#why-a-run-failed) shows; fix the cause, then plan and apply again |
| `Building block run is waiting for input or approval` | an approval gate or a missing operator input stops the run; the apply does not wait for it | read the preflight and ask the user to approve, as [Approval gates and preflight runs](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#approval-gates-and-preflight-runs) shows |
| `Building block run has not finished` | the timeout ended while the run was pending or in progress | wait for it, as [Waiting for a run](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#waiting-for-a-run) shows |
| error `meshStack did not start a run for the updated building block` | meshStack had not started the run after a minute | wait for the run, then apply again |

A human approves a gated run in meshPanel. Never try to approve it yourself. After a run ended
outside the apply, `tofu plan` reads the block's new status. `status.action` names what a block
needs next, such as `APPROVE_RUN`, `PROVIDE_INPUT` or `REVIEW_ERRORS`.

## Import existing objects

Import a building block or a definition by its uuid with an `import` block, as
[import-by-string-id.tf](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/examples/resources/meshstack_building_block/import-by-string-id.tf)
shows, and let the plan write the resource:

```sh
tofu plan -generate-config-out=generated.tf
```

Review the generated resource and move it into your configuration. Plan again until the plan shows
only the import, then apply.

- Find the uuid with the data sources above, or with the CLI.
- A generated sensitive input has `secret_value = null`. Replace it with an ephemeral variable as
  [Add a building block](#add-a-building-block) shows, or the plan fails.
- Leave `content_hash` out of an imported block until after the first apply. Otherwise that apply
  runs the block again.
- Only the workspace that owns a definition can manage it.
- The registry page of each resource names its import id.

To move a deprecated `meshstack_building_block_v2` or `meshstack_buildingblock` to
`meshstack_building_block`, rewrite the resource block for the new schema and add:

```hcl
moved {
  from = meshstack_building_block_v2.repo
  to   = meshstack_building_block.repo
}
```

## Always plan before apply

```sh
tofu plan -out=tfplan
tofu apply tfplan
```

- Run `plan` and show the user the plan before every `apply`: the `Plan:` line, and each object
  that is created, changed, replaced or destroyed. Apply only after the user agreed, and apply the
  saved plan, so the apply does exactly what the user saw.
- Ask before anything that destroys or replaces (`-`, `-/+`, `must be replaced`). Destroying a
  building block starts its destroy run, which removes its cloud resources.
- `plan` only reads from meshStack. Run it as often as you need.
- A human approves gated runs and definition releases in meshPanel.
- Keep secrets out of the code and the state: the API key secret in `MESHSTACK_API_SECRET`, a
  sensitive input from an ephemeral variable. Never commit a state file or a plan file.

## Run a config against a dev build

This section is for contributors. Run a config with the provider you built, to reproduce a
provider bug or a failing acceptance test, or to grow a test's config into a demo. It needs a clone
of this repository and an API key for a meshStack: a local dev stack, or a dev or sandbox
meshStack. The config makes real changes there, so never point it at a production meshStack.
[DEVELOPMENT.md](https://raw.githubusercontent.com/meshcloud/terraform-provider-meshstack/main/DEVELOPMENT.md#backends--authentication)
says where the endpoint and the key come from.

Build the provider into the repository root once, and point a CLI config of its own at it, so that
your global `~/.terraformrc` stays unchanged:

```sh
task build
cat > "$HOME/.terraformrc.dev" <<EOF
provider_installation {
  dev_overrides {
    "meshcloud/meshstack" = "$PWD"
  }
  direct {}
}
EOF
```

Then, in every shell that runs the config:

```sh
export TF_CLI_CONFIG_FILE=$HOME/.terraformrc.dev
set -a && source .env && set +a
cd scratch/TestAccBuildingBlock/01_workspace_lifecycle/step01
tofu plan
tofu apply
tofu destroy
```

- `.env` sets `MESHSTACK_ENDPOINT`, `MESHSTACK_API_KEY` and `MESHSTACK_API_SECRET`. `tofu destroy`
  removes the meshStack objects again when you are done.
- With a dev override, `tofu` warns and needs no `init`. Run `task build` again after every change
  to the provider.
- Take the config from a test, as [Get a working config](#get-a-working-config) shows, or write a
  `main.tf` into a folder such as `scratch/repro/` and add the `provider.tf` the dump writes.
- `TF_LOG_PROVIDER=debug tofu apply` prints the provider's log. To use a debugger, build with
  `go build -gcflags="all=-N -l"` and attach delve to the running provider process.
- `TF_ACC_PERSIST_WORKING_DIR=1` keeps the working folder of a test, but the test runs the provider
  inside its own process, so that folder does not run on its own. The dump does.
- To test a `terraform` definition without real infrastructure, point its implementation at the
  NoOp module: repository `https://github.com/meshcloud/meshstack-hub.git`, path
  `modules/meshstack/noop/buildingblock`. It declares every input and output type and creates
  nothing. Keep `draft = true` while you change it, as [Release a version](#release-a-version)
  explains.
- `scratch/` is ignored by git, and the state lives there too, so both are lost with the folder. Move
  a config that is worth keeping out of `scratch/`, for example into `examples/`, and give it a
  remote state backend.
