# resource-test-1.tf assigning `region` as a customer input. The definition declares it STATIC, so
# this has to be rejected — by the backend, which is the only side that knows the assignment type.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest.uuid }

    display_name = var.bb_display_name
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      name = {
        value = jsonencode(var.bb_name)
      }
      size = {
        value = jsonencode(16)
      }
      environment = {
        value = jsonencode(var.bb_environment)
      }
      region = {
        value = jsonencode("eu-central-1")
      }
    }
  }

  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }

  lifecycle {
    postcondition {
      condition     = !contains(["FAILED", "ABORTED"], self.status.status)
      error_message = "Building block ${self.metadata.uuid} is ${self.status.status}. See its run in meshPanel."
    }
  }
}
