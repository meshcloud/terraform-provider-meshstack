resource "meshstack_building_block_v2" "example_tenant" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.bb_v2_tenant_bbd.version_latest

    display_name = "my-tenant-building-block"
    target_ref   = meshstack_tenant.example.ref

    inputs = {
      name        = { value_string = "my-name" }
      size        = { value_int = 16 }
      environment = { value_single_select = "dev" }
    }
  }
}
