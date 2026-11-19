resource "meshstack_project_tag" "example" {
  metadata = {
    workspace_identifier = data.meshstack_project.example.metadata.owned_by_workspace
    project_identifier   = data.meshstack_project.example.metadata.name
    key                  = data.meshstack_tag_definition.cost_center.spec.key
  }
  spec = {
    values = ["12345"]
  }
}
