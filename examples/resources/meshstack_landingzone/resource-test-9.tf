# The landing zone with a default quota, for the case where the effective quotas legitimately differ
# from what the tenant requested: meshStack merges this default in, so status.applied_quotas is a
# strict superset of spec.requested_quotas.

resource "meshstack_landingzone" "example" {
  metadata = {
    name               = "test-lz-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
    tags               = {}
  }

  spec = {
    display_name                  = "My Custom Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"

    platform_ref                  = meshstack_platform.example_custom.ref
    mandatory_building_block_refs = [meshstack_building_block_definition.mandatory_bbd.ref]

    quotas = var.lz_memory_default > 0 ? [{ key = "limits.memory", value = var.lz_memory_default }] : []

    platform_properties = {
      custom = {}
    }
  }
}
