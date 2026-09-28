# Exactly one tracked override: approval is derived (NONE, display_name equal to the input's) and
# prunes away, so an import must reproduce this subset and nothing more.
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
      region = { assignment_type = "SUMMARY" }
    }
  }
}
