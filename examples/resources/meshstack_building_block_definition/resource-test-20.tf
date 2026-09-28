# The manual definition switched to terraform with an approval gate enabled. The plan-time policy
# validators read the implementation the configuration asks for, so this passes them and reaches
# Update — where a released version's immutable version_spec is what must be reported.
# An example for a manual implementation, showing a sparse output override.
resource "meshstack_building_block_definition" "example_03_manual" {
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

    # null means the attribute is absent; "" means an explicitly empty template. meshStack treats them
    # alike, Terraform does not, so a case walks both.
    display_name_template = var.display_name_template
  }

  version_spec = {
    draft = var.draft

    inputs = {
      approval_required = {
        display_name    = "Approval Required"
        type            = "BOOLEAN"
        assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT"
      }
      resource_url = {
        display_name    = "Resource URL"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
      }
    }

    # Manual building blocks are special: the backend derives one output per input, and each output's
    # `type` is always taken from its input (it must NOT be set here). `version_spec.outputs` is therefore
    # a SPARSE OVERRIDE — declare only the outputs you want to customize and leave the rest to be derived.
    # For a declared output you may set `assignment_type` (to mark how the value is used) and, optionally,
    # `display_name` and `display_order`. Omit the attribute entirely (or set `{}`) to customize nothing.
    # The manual example's sparse overrides have no meaning for terraform.
    outputs = {}
  }
}
