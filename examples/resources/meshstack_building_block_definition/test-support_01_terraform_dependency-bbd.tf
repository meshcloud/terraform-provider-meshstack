# A minimal manual BBD used as a dependency for the terraform example
resource "meshstack_building_block_definition" "other" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Dependency Building Block"
    description  = "A building block that is used as a dependency"
  }

  version_spec = {
    draft = false
    implementation = {
      manual = {}
    }
  }
}
