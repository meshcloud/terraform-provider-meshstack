resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Test BB v3 Payment Method Input"
    description  = "A building block definition that asks for one of the workspace's Payment Methods"
  }

  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name           = "Name"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      payment_method = {
        display_name           = "Payment Method"
        description            = "The Payment Method this building block books its cost on"
        type                   = "CODE"
        assignment_type        = "PAYMENT_METHOD"
        updateable_by_consumer = true
      }
    }

    implementation = {
      manual = {}
    }
  }
}
