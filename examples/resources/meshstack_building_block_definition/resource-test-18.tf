# The github definition with the policy attributes exposed as variables.
# An example for github_workflows implementation with required attributes only
resource "meshstack_building_block_definition" "example_02_github_workflows" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    # The policies a case dials. github_workflows has no dry run, so it supports drift reconciliation
    # only with automatic approval and no approval gate at all; terraform supports every policy.
    approval_policies = var.approval_policies
    schedule          = var.schedule
    display_name      = "Example Building Block"
    description       = var.description
  }

  version_spec = {
    draft = var.draft

    inputs = {
      workflow_ref = {
        display_name    = "Workflow Reference"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    deletion_mode = "PURGE"
    implementation = {
      github_workflows = {
        repository      = "example/building-block"
        branch          = "main"
        apply_workflow  = "apply.yml"
        integration_ref = meshstack_integration.github.ref
        # Optional flags, default false
        async                 = true
        omit_run_object_input = true
      }
    }

    outputs = {
      workflow_run_url = {
        display_name    = "Workflow Run URL"
        type            = "STRING"
        assignment_type = "RESOURCE_URL"
      }
    }
  }
}
