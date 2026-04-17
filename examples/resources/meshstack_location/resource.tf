resource "meshstack_location" "example" {
  metadata = {
    name               = "my-location"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "My Cloud Location"
    description  = "A location for managing cloud resources"
  }
}
