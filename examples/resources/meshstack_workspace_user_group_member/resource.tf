resource "meshstack_workspace_user_group_member" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace_user_group.example.metadata.owned_by_workspace
    group_identifier     = data.meshstack_workspace_user_group.example.metadata.name
    username             = data.meshstack_user.example.metadata.name
  }
}
