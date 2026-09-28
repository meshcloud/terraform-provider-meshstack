data "meshstack_platform_type" "example" {
  metadata = {
    name               = meshstack_platform_type.example.metadata.name
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }
}
