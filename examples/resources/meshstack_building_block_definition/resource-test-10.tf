# The starting override set of the override-lifecycle case.
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
      approval = { assignment_type = "SUMMARY" }
      region   = { display_name = "Region Output" }
    }
  }
}
