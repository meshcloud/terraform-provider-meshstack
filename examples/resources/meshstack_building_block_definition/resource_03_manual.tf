# An example for manual implementation with required attributes only
resource "meshstack_building_block_definition" "example_03_manual" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Example Building Block"
    description  = "An example building block definition"
  }

  version_spec = {
    draft = true

    inputs = {
      approval_required = {
        display_name    = "Approval Required"
        type            = "BOOLEAN"
        assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT"
      }
    }

    implementation = {
      manual = {}
    }

    # Output keys must match with inputs, as the backend copies over inputs to outputs
    outputs = {
      approval_required = {
        display_name    = "Approval Required"
        type            = "BOOLEAN"
        assignment_type = "NONE"
      }
    }
  }
}
