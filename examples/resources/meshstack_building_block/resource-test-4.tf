resource "meshstack_building_block" "example_tenant" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example_tenant.version_latest.uuid }

    display_name = "my-tenant-building-block"
    target_ref   = meshstack_tenant.example.ref

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

  # create/update wait for the building block run to reach a terminal state; delete waits for
  # deprovisioning.
  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }
}
