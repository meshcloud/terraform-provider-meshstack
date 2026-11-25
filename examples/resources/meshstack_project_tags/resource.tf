resource "meshstack_project_tags" "example" {
  project_ref = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    name               = data.meshstack_project.example.metadata.name
  }
  tags = {
    (data.meshstack_tag_definition.cost_center.spec.key) = ["12345"]
  }
}
