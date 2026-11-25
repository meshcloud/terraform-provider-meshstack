resource "meshstack_project_tag" "example" {
  project_ref = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    name               = data.meshstack_project.example.metadata.name
  }
  key    = data.meshstack_tag_definition.cost_center.spec.key
  values = ["12345"]
}
