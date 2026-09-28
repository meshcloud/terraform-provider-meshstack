resource "meshstack_workspace_tags" "example" {
  metadata = {
    workspace_identifier = meshstack_workspace.example.metadata.name
  }
  spec = {
    tags = {
      (meshstack_tag_definition.first_tag.spec.key) = ["12345"]
    }
  }
}
