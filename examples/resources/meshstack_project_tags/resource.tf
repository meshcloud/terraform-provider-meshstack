resource "meshstack_project_tags" "example" {
  metadata = {
    workspace_identifier = data.meshstack_project.example.metadata.owned_by_workspace
    project_identifier   = data.meshstack_project.example.metadata.name
  }
  spec = {
    tags = {
      (data.meshstack_tag_definition.cost_center.spec.key) = ["12345"]
    }
  }
}
