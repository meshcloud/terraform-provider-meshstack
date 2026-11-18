resource "meshstack_workspace_user_group" "example" {
  metadata = {
    name               = "my-user-group"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    display_name = "My User Group"

    # Usernames, not email addresses. Omit members to manage them elsewhere,
    # for example with meshstack_workspace_user_group_member.
    members = [data.meshstack_user.example.metadata.name]
  }
}
