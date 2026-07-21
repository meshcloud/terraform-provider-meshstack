# An example for a manual implementation, showing a sparse output override.
resource "meshstack_building_block_definition" "example_03_manual" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
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
      resource_url = {
        display_name    = "Resource URL"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    implementation = {
      manual = {}
    }

    # Manual building blocks are special: the backend derives one output per input, and each output's
    # `type` is always taken from its input (it must NOT be set here). `version_spec.outputs` is therefore
    # a SPARSE OVERRIDE — declare only the outputs you want to customize and leave the rest to be derived.
    # For a declared output you may set `assignment_type` (to mark how the value is used) and, optionally,
    # `display_name` and `display_order`. Omit the attribute entirely (or set `{}`) to customize nothing.
    outputs = {
      # Mark the operator-provided URL as a resource link and give it a friendlier label and position.
      resource_url = {
        assignment_type = "RESOURCE_URL"
        display_name    = "Provisioned Resource"
        display_order   = 1
      }
      # approval_required is not listed here, so its output is derived as-is.
    }
  }
}
