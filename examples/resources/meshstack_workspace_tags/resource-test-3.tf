resource "meshstack_workspace_tags" "example" {
  metadata = {
    workspace_identifier = meshstack_workspace.example.metadata.name
  }
  spec = {
    # A declared key with no values must converge. The API returns an entry for every defined tag
    # property — an empty list when unset — and reconcileTags mirrors tracked keys verbatim, so the
    # key stays in state instead of being dropped on every refresh.
    tags = {
      (meshstack_tag_definition.first_tag.spec.key) = []
    }
  }
}
