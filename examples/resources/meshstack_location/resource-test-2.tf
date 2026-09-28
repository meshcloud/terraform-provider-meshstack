resource "meshstack_location" "example" {
  metadata = {
    name               = "my-location-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "My Updated Location"
    description  = "A location for managing cloud resources"
  }
}
