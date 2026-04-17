resource "meshstack_landingzone" "example" {
  metadata = {
    name               = "my-landing-zone-custom"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    tags               = {}
  }

  spec = {
    display_name                  = "My Custom Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"

    platform_ref = data.meshstack_platform.example.ref

    platform_properties = {
      // Nothing to be specified for custom platforms, but the block must be present.
      custom = {}
    }
  }
}
