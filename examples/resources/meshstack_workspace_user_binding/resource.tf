resource "meshstack_workspace_user_binding" "example" {
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
    name = "user@meshcloud.io"
  }
}
