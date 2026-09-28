# The v1 side of the v1->v3 migration, on the dedicated migration definition rather than the v2
# tenant showcase: the legacy resource's input shape cannot carry sensitive values.

resource "meshstack_buildingblock" "my_buildingblock" {
  metadata = {
    definition_uuid    = meshstack_building_block_definition.example_tenant.ref.uuid
    definition_version = meshstack_building_block_definition.example_tenant.version_latest.number

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
