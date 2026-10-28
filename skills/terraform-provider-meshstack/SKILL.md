---
name: terraform-provider-meshstack
description: >
  Manage meshStack building block definitions and building blocks as code with the meshStack
  Terraform/OpenTofu provider. Covers login, looking up objects, creating and releasing a
  definition, adding, changing, upgrading and deleting a building block, waiting for runs and
  approvals, and importing what exists. Use when asked to write or change `meshstack_*` HCL, to
  order or update a building block through Terraform, to publish a definition version, or to bring
  meshStack objects under Terraform.
---

# meshStack Terraform provider

This skill is for agents and people who **use** the provider. It lives in the provider repository,
so it changes together with the resources.

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

## Log in

Act as the user, through their own meshStack CLI browser login. meshStack then records the user as
the author of every change. The CLI skill shows
[how an agent starts that login](https://raw.githubusercontent.com/meshcloud/meshstack-cli/main/skills/meshstack-cli/SKILL.md#logging-in-as-an-agent).
The provider uses the stored login, renews it, and never opens a browser. When the login has
expired, the provider fails, and the user has to log in again.

```hcl
terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  profile   = "<profile>"
  workspace = "<workspace>"
}
```

- A browser login needs `workspace`, or `MESHSTACK_WORKSPACE`. The provider ignores the default
  workspace of the profile, so the configuration acts in the same workspace for everyone who
  applies it. Use the identifier meshPanel shows, such as `devops-platform`.
- The login acts with the user's role in that workspace. Set it to the workspace the objects belong
  to: the platform team's workspace for a definition, the building block's workspace for a block.
- For several workspaces, write one provider block per workspace with an `alias`, and set
  `provider = meshstack.<alias>` on each resource.
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
uuids into the configuration:

```hcl
data "meshstack_workspace" "team" {
  metadata = {
    name = "<workspace>"
  }
}

data "meshstack_building_block_definition" "repo" {
  spec = {
    display_name = "<definition name as the meshPanel marketplace shows it>"
  }
}

data "meshstack_tenants" "dev" {
  workspace = "<workspace>"
  project   = "<project>"
  platform  = "<platform>.<location>"
}

data "meshstack_payment_method" "team" {
  metadata = {
    name               = "<payment method>"
    owned_by_workspace = "<workspace>"
  }
}

data "meshstack_building_blocks" "failed" {
  workspace_identifier = "<workspace>"
  health_status        = ["FAILED"]
}
```

- `data.meshstack_workspace.team.ref` is the `target_ref` of a workspace-level building block, and
  `one(data.meshstack_tenants.dev.tenants).ref` the one of a tenant-level block.
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

A definition belongs to the platform team's workspace. It starts as a draft:

```hcl
resource "meshstack_building_block_definition" "repo" {
  metadata = {
    owned_by_workspace = "<platform team workspace>"
  }

  spec = {
    display_name = "GitHub Repository"
    description  = "A GitHub repository for one application team."
    target_type  = "WORKSPACE_LEVEL"
  }

  version_spec = {
    draft = true

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/<org>/<repo>.git"
        repository_path   = "modules/github-repository"
        ref_name          = "v1.0.0"
      }
    }

    inputs = {
      repo_name = {
        display_name    = "Repository name"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
      visibility = {
        display_name           = "Visibility"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        selectable_values      = ["private", "internal"]
        default_value          = jsonencode("private")
        updateable_by_consumer = true
      }
    }

    outputs = {
      repo_url = {
        display_name    = "Repository"
        type            = "STRING"
        assignment_type = "RESOURCE_URL"
      }
    }
  }
}
```

- `default_value` and `argument` are `jsonencode`d to match `type`, also for a string.
- A `TENANT_LEVEL` definition also needs `spec.supported_platforms`. `target_type` cannot change:
  a changed `target_type` replaces the definition.
- `manual`, `github_workflows`, `gitlab_pipeline` and `azure_devops_pipeline` are the other
  implementations. Approval gates (`spec.approval_policies`) and drift detection need `terraform`.
  See the [registry page](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs/resources/building_block_definition)
  for every input and output attribute.
- Only the owning workspace can order a draft. Test the draft there with a building block whose
  `building_block_definition_version_ref` is `meshstack_building_block_definition.repo.version_latest`.
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

```hcl
resource "meshstack_building_block" "repo" {
  spec = {
    building_block_definition_version_ref = data.meshstack_building_block_definition.repo.version_latest_release
    display_name                          = "checkout-service repository"
    target_ref                            = data.meshstack_workspace.team.ref

    inputs = {
      repo_name = {
        value = jsonencode("checkout-service")
      }
    }
  }

  lifecycle {
    postcondition {
      condition     = self.status.succeeded
      error_message = "Building block ${self.metadata.uuid} did not succeed: health status ${self.status.health_status}, action ${self.status.action}."
    }
  }
}
```

- Set the inputs the version declares as `USER_INPUT` or `PAYMENT_METHOD`. meshStack and the
  platform team fill in the others. `inputs` is required, so write `inputs = {}` when you set none.
- Every `value` is `jsonencode`d, also a string. A `PAYMENT_METHOD` value is
  `jsonencode(data.meshstack_payment_method.team.ref)`.
- Keep the postcondition. Without it, a failed run is only a warning, and the apply looks
  successful.
- `version_latest_release` is null until the definition has a released version.

A sensitive input takes `sensitive` instead of `value`. Feed it from an ephemeral variable, so the
value is neither in the code nor in state, and change `secret_version` to send a new value. Set
`TF_VAR_deploy_token` for the plan and again for the apply, because a saved plan does not keep an
ephemeral value. This needs Terraform or OpenTofu 1.11 or later.

```hcl
variable "deploy_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "meshstack_building_block" "pipeline" {
  spec = {
    building_block_definition_version_ref = data.meshstack_building_block_definition.repo.version_latest_release
    display_name                          = "checkout-service pipeline"
    target_ref                            = data.meshstack_workspace.team.ref

    inputs = {
      deploy_token = {
        sensitive = {
          secret_value   = var.deploy_token
          secret_version = "1"
        }
      }
    }
  }
}
```

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

Import a building block or a definition by its uuid:

```hcl
import {
  id = "<building block uuid>"
  to = meshstack_building_block.repo
}
```

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
