resource "meshstack_building_block_v2" "example_tenant" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = one(data.meshstack_building_block_definitions.example.building_block_definitions).version_latest

    display_name = "my-tenant-building-block"
    target_ref   = one(data.meshstack_tenants.example.tenants).ref

    inputs = {
      name        = { value_string = "my-name" }
      size        = { value_int = 16 }
      environment = { value_single_select = "dev" }
    }
  }
}
