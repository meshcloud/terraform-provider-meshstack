# The mirror of resource-test-10.tf: kind = meshWorkspace addressed by uuid, where a workspace is
# addressed by name.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest.uuid }

    display_name = var.bb_display_name
    target_ref   = { kind = "meshWorkspace", uuid = "00000000-0000-0000-0000-000000000000" }

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
