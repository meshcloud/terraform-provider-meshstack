resource "meshstack_workspace_tag" "first" {
  metadata = {
    workspace_identifier = meshstack_workspace.example.metadata.name
    key                  = meshstack_tag_definition.first_tag.spec.key
  }
  spec = {
    values = ["12345"]
  }
}
