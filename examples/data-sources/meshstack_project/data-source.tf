data "meshstack_project" "example" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
}
