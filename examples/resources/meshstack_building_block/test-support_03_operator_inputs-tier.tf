# test-support_03_operator_inputs.tf plus a second, *defaulted* platform operator input. Swapping it
# in edits the same definition, so it is the v2 of that definition: re-drafted while
# version_latest_release still resolves to v1, then released. The backend applies `tier`'s default on
# upgrade, so a block that never supplies it still reaches SUCCEEDED rather than parking.
resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "Test BB v3 Operator Inputs"
    description      = "A building block definition with user and operator inputs"
    run_transparency = true
  }

  version_spec = {
    draft = var.bbd_draft

    inputs = {
      name = {
        display_name           = "Name"
        description            = "Name of the resource"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      size = {
        display_name           = "Size"
        description            = "A platform operator input"
        type                   = "INTEGER"
        assignment_type        = "PLATFORM_OPERATOR_MANUAL_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        description            = "Target environment"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        selectable_values      = ["dev", "staging", "prod"]
      }
      tier = {
        display_name           = "Tier"
        description            = "A defaulted platform operator input"
        type                   = "STRING"
        assignment_type        = "PLATFORM_OPERATOR_MANUAL_INPUT"
        default_value          = jsonencode("bronze")
        updateable_by_consumer = true
      }
    }

    implementation = {
      manual = {}
    }

  }
}
