resource "meshstack_building_block_v3" "example_workspace" {
  # Last resort option for stuck deletions. Prefer regular delete behavior.
  # purge_on_delete = true

  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = one(data.meshstack_building_block_definitions.example.building_block_definitions).version_latest

    display_name = "my-workspace-building-block"
    target_ref   = data.meshstack_workspace.example.ref

    inputs = {
      name = {
        value = "my-name"
      }
      size = {
        value = jsonencode(16)
      }
      environment = {
        value = "dev"
      }
    }
  }
}
