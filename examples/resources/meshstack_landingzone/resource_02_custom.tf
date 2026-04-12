resource "meshstack_landingzone" "example" {
  metadata = {
    name               = "my-landing-zone-custom"
    owned_by_workspace = "my-workspace"
    tags               = {}
  }

  spec = {
    display_name                  = "My Custom Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"

    platform_ref = {
      // UUID of an existing custom platform.
      uuid = "7035ad04-f912-44d5-98ce-ddcc2cf84b10"
    }

    platform_properties = {
      // Nothing to be specified for custom platforms, but the block must be present.
      custom = {}
    }
  }
}
