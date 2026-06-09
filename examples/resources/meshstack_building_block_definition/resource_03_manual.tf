# An example for manual implementation with required attributes only
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
    }

    implementation = {
      manual = {}
    }

    # Outputs are omitted for manual building blocks: the backend derives them from the inputs
    # (one output per input), so version_spec.outputs is computed and must not be set here.
  }
}
