resource "meshstack_building_block" "example_workspace" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = one(data.meshstack_building_block_definitions.example.building_block_definitions).version_latest

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

  # The provider only reports a run that does not succeed: FAILED and ABORTED are warnings and the next
  # plan runs the building block again, a WAITING_FOR_* one stays parked until someone acts in meshPanel.
  # This postcondition is what fails the apply instead. It is checked after the building block is written
  # to state, so it does not taint the block and the next apply still runs it again.
  #
  # FAILED and ABORTED mean the run broke. The WAITING_FOR_* statuses do not: the block is waiting for an
  # input or an approval, which is why they are not listed here. Use
  # `self.status.status == "SUCCEEDED"` instead where the apply must go red until the block is finished —
  # a starterkit ordering blocks for a tenant, for example.
  lifecycle {
    postcondition {
      condition     = !contains(["FAILED", "ABORTED"], self.status.status)
      error_message = "Building block ${self.metadata.uuid} is ${self.status.status}. See its run in meshPanel."
    }
  }
}
