resource "meshstack_building_block" "example_workspace" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = data.meshstack_building_block_definition.example.version_latest

    display_name = "my-workspace-building-block"
    target_ref   = data.meshstack_workspace.example.ref

    inputs = {
      name = {
        value = jsonencode("my-name")
      }
      size = {
        value = jsonencode(16)
      }
      environment = {
        value = jsonencode("dev")
      }
      payment_method = {
        value = jsonencode(data.meshstack_payment_method.example.ref)
      }
    }

    # Building blocks can depend on each other: a parent's outputs feed this block's inputs.
    # Reference a parent by its computed `ref`.
    # parent_building_block_refs = [meshstack_building_block.parent.ref]
  }

  # Purging is a last resort option for stuck deletions. Prefer regular delete behavior.
  # purge_on_delete = true

  # create/update wait for the building block run to reach a terminal state; delete waits for
  # deprovisioning. Tune to your runner's typical run duration (defaults to 30m if unset).
  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }

  # The provider only warns about a run that does not succeed; this fails the apply without tainting the block.
  lifecycle {
    postcondition {
      condition     = self.status.succeeded
      error_message = "Building block ${self.metadata.uuid} did not succeed (health status ${self.status.health_status}, action ${self.status.action}). See its run in meshPanel."
    }
  }
}
