resource "meshstack_location" "example" {
  metadata = {
    name = "my-location"
  }

  spec = {
    display_name = "My Cloud Location"
    description  = "A location for managing cloud resources"
  }
}
