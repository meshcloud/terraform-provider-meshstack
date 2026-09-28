# resource-test-1.tf with the whole version object behind the ref plus an explicit content_hash, for
# the steps that walk a definition content change. content_hash is provider-only and tracks the
# version's content, so changing it must trigger a rerun even though the version uuid is unchanged.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = merge(meshstack_building_block_definition.example.version_latest, { content_hash = var.bb_content_hash })

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
