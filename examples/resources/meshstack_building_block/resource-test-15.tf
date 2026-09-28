# A block the consumer workspace creates itself, through the meshstack-other provider alias backed by
# its own workspace-scoped key. Whether it may change a given input is then up to what the
# definition marks updateable by a consumer.

resource "meshstack_building_block" "example_workspace" {
  provider = meshstack-other

  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

    display_name = var.bb_display_name
    target_ref   = meshstack_workspace.other.ref

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
    }
  }

  # The other provider needs the key for teardown, so the block has to go first.
  depends_on = [meshstack_api_key.example]

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
