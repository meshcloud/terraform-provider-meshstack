data "meshstack_project" "example" {
  metadata = {
    name               = meshstack_project.example.metadata.name
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }
}
