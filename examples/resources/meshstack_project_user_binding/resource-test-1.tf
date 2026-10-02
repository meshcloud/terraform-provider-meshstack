resource "meshstack_project_user_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Project Reader"
  }

  target_ref = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
    name               = meshstack_project.example.metadata.name
  }

  subject = {
    name = "user@meshcloud.io"
  }
}
