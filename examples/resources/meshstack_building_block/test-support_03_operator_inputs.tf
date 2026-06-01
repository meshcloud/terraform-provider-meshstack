resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name     = "Test BB v3 Operator Inputs"
    description      = "A building block definition with user and operator inputs"
    run_transparency = true
  }

  # Reuses the same input keys as the workspace example (resource_01_workspace.tf) so that example is
  # 1:1 reusable, but changes their behaviour: `size` is a platform-operator input (settable only by an
  # operator/admin), while `name`/`environment` stay regular consumer inputs.
  version_spec = {
    draft = false

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
    }

    implementation = {
      manual = {}
    }

  }
}
