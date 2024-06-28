resource "meshstack_buildingblock" "my_buildingblock" {
  metadata = {
    definition_uuid    = "f012248e-dda9-4763-8706-641a35de6c62"
    definition_version = 1
    tenant_identifier  = "my-workspace.my-project-dev.my-platform.my-location"
  }

  spec = {
    display_name = "my-buildingblock"

    inputs = {
      name = { value_string = "my-name" }
      size = { value_int = 16 }
    }
  }
}
