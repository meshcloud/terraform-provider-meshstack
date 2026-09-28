# The v3 side of the v1->v3 migration. The resource label matches the `moved` block target in
# test-support_moved_from_v1.tf, and the inputs mirror the migration definition, which carries no
# sensitive inputs because the legacy v1 resource cannot express them.

resource "meshstack_building_block" "example_tenant" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example_tenant.version_latest

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

  # Bounded waits so tests fail reasonably fast (vs the 30m default) while tolerating a busy runner.
  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }
}
