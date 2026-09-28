# resource-test-1.tf wiring the whole version object rather than only its uuid, which is what a
# configuration written against a definition in the same state normally does. The ref then carries
# the definition's own content_hash.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

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
