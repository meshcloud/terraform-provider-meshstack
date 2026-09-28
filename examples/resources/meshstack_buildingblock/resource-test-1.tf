resource "meshstack_buildingblock" "my_buildingblock" {
  metadata = {
    definition_uuid    = meshstack_building_block_definition.bb_v2_tenant_bbd.ref.uuid
    definition_version = meshstack_building_block_definition.bb_v2_tenant_bbd.version_latest.number

    # The v1 resource addresses its tenant by the composite identifier rather than a ref.
    tenant_identifier = "${meshstack_tenant.example.metadata.owned_by_workspace}.${meshstack_tenant.example.metadata.owned_by_project}.${meshstack_platform.example_custom.identifier}"
  }

  spec = {
    display_name = "my-buildingblock"

    inputs = {
      name        = { value_string = "my-name" }
      size        = { value_int = 16 }
      environment = { value_single_select = "dev" }
    }
  }
}
