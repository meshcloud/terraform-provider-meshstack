resource "meshstack_workspace_tags" "example" {
  metadata = {
    # The workspace example's step is deliberately not applied alongside this one, so nothing creates
    # the workspace it names.
    workspace_identifier = "does-not-exist-workspace"
  }
  spec = {
    tags = {
      (meshstack_tag_definition.first_tag.spec.key) = ["12345"]
    }
  }
}
