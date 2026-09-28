# resource-test-5.tf plus a parent. The uuid is synthetic: this file exists to observe the plan
# decision that adding a parent without a version upgrade forces a replacement, and the real backend
# rejects the uuid before that plan can be applied.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = merge(meshstack_building_block_definition.example.version_latest, { content_hash = var.bb_content_hash })

    display_name = var.bb_display_name
    target_ref   = meshstack_workspace.example.ref

    parent_building_block_refs = [{ uuid = "11111111-1111-1111-1111-111111111111" }]

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
