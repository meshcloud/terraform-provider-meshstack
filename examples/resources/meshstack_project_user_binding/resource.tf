resource "meshstack_project_user_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Project Reader"
  }

  target_ref = {
    owned_by_workspace = "my-customer"
    name               = "my-project"
  }

  subject = {
    name = "user@meshcloud.io"
  }
}
