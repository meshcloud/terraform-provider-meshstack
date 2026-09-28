# approval's override dropped: the stateful backend would otherwise preserve it, so this
# exercises the full-set send that resets it. region is renamed at the same time.
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
      region = { display_name = "Region Renamed" }
    }
  }
}
