# The same definition with the implementation switched to terraform, which is what makes the
# approval gates and drift detection writable at all.
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
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
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
