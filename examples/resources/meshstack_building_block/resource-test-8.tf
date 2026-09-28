# resource-test-7.tf declaring only the operator input, so the user inputs are left to whoever owns
# them. What this configuration omits must be preserved server-side and surfaced read-only in
# all_inputs — not dropped, and not drift.

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

    display_name = var.bb_display_name
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      size = {
        value = jsonencode(16)
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
