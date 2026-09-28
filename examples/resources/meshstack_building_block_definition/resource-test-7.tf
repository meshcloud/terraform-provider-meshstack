# Manual definition with no output overrides at all: the backend derives one output per input and
# the provider prunes every non-override, so the tracked set is the empty map.
resource "meshstack_building_block_definition" "example_03_manual" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Example Building Block"
    description  = var.description
  }

  version_spec = {
    draft = var.draft

    inputs = {
      approval = { display_name = "Approval", type = "BOOLEAN", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT" }
      region   = { display_name = "Region", type = "SINGLE_SELECT", assignment_type = "USER_INPUT", selectable_values = ["eu", "us"] }
    }

    implementation = {
      manual = {}
    }

    outputs = {
    }
  }
}
