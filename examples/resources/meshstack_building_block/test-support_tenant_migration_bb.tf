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

  # Short waits so tests fail fast instead of polling the 30m default if a run hangs.
  timeouts = {
    create = "30s"
    update = "30s"
    delete = "30s"
  }
}
