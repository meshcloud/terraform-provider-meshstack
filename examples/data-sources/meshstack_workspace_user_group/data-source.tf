data "meshstack_workspace_user_group" "example" {
  metadata = {
    name               = "my-user-group"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
}
