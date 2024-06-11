data "meshstack_project" "example" {
  metadata = {
    name               = "my-project-identifier"
    owned_by_workspace = "my-workspace-identifier"
  }
}
