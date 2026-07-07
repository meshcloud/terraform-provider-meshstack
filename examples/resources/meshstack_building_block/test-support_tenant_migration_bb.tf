# v3 tenant building block for the v1->v3 migration test (04_tenant_moved_from_v1). The resource
# label (example_tenant) matches the `moved` block target in test-support_moved_from_v1.tf. The
# version ref and target ref are placeholders overridden by the test; inputs mirror the migration
# BBD (no sensitive inputs — see test-support_tenant_migration_bbd.tf).
resource "meshstack_building_block" "example_tenant" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-tenant-building-block"
    target_ref                            = { kind = "meshTenant", uuid = "placeholder" }

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
