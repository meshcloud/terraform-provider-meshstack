# kind = meshTenant addressed by name. A tenant is addressed by uuid, so the provider's own validator
# has to reject this before any backend call — which is why this case runs in both modes.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest.uuid }

    display_name = var.bb_display_name
    target_ref   = { kind = "meshTenant", name = "some-workspace" }

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
