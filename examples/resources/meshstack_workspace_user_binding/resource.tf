resource "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Project Reader"
  }

  target_ref = {
    name = "my-project"
  }

  subject = {
    name = "user@meshcloud.io"
  }
}
