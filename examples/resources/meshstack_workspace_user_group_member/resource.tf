resource "meshstack_workspace_user_group_member" "example" {
  group_ref = meshstack_workspace_user_group.example.ref
  user_ref  = data.meshstack_user.example.ref
}
