resource "meshstack_workspace_group_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = "my-workspace"
  }

  subject = {
    name = "my-user-group"
  }
}
