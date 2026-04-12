resource "meshstack_building_block_v2" "example_tenant" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = {
      uuid = "00000000-0000-0000-0000-000000000000"
    }

    display_name = "my-tenant-building-block"
    target_ref = {
      kind       = "meshTenant"
      identifier = "my-workspace.my-project.my-location.my-platform"
    }

    inputs = {
      name        = { value_string = "my-name" }
      size        = { value_int = 16 }
      environment = { value_single_select = "dev" }
    }
  }
}
