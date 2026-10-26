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

  # Optional. If omitted, the binding never expires.
  expiry_date = "2099-12-31"
}
