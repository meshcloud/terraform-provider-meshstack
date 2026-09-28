resource "meshstack_landingzone" "example" {
  metadata = {
    name               = "test-lz-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
    tags               = {}
  }

  spec = {
    display_name                  = "Updated Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"

    # The platform's computed `ref` (kind + uuid) is assigned inline — platform_ref accepts the full
    # ref object, so no explicit uuid mapping is needed.
    platform_ref                  = meshstack_platform.example_custom.ref
    mandatory_building_block_refs = [meshstack_building_block_definition.mandatory_bbd.ref]

    platform_properties = {
      // Nothing to be specified for custom platforms, but the block must be present.
      custom = {}
    }
  }
}
