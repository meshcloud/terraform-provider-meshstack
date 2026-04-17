data "meshstack_project" "example" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = "my-workspace"
  }
}
