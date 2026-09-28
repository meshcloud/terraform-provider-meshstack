# A block in the consumer workspace, created by the definition owner across the workspace boundary
# and pinned to the definition's latest *released* version so a later release can drive an in-place
# upgrade. The operator input the definition declares is left out here, which parks the block in
# WAITING_FOR_OPERATOR_INPUT until someone supplies it.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest_release.uuid }

    display_name = var.bb_display_name
    target_ref   = meshstack_workspace.other.ref

    inputs = {
      name = {
        value = jsonencode(var.bb_name)
      }
      environment = {
        value = jsonencode(var.bb_environment)
      }
    }
  }

  # Keeps the minted key alive until after the block is torn down.
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
