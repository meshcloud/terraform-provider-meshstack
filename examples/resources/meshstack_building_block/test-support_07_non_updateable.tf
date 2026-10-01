resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name     = "Test BB v3 Non-Updateable Input"
    description      = "A building block definition with a non-updateable consumer input"
    run_transparency = true
  }

  # Reuses the same input keys as the workspace example (resource_01_workspace.tf) so that example is
  # 1:1 reusable, but marks `environment` and `payment_method` as not updateable by the consumer —
  # changing them from a consumer-scoped key must be rejected.
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
        description            = "Size of the resource"
        type                   = "INTEGER"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        description            = "A user input not updateable by consumer"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = false
        selectable_values      = ["dev", "staging", "prod"]
      }
      payment_method = {
        display_name           = "Payment Method"
        type                   = "CODE"
        assignment_type        = "PAYMENT_METHOD"
        updateable_by_consumer = false
      }
    }

    implementation = {
      manual = {}
    }

  }
}
