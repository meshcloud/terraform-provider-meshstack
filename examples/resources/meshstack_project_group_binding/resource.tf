resource "meshstack_project_group_binding" "example" {
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
    name = "my-user-group"
  }
}
