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
}
