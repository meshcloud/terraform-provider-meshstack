resource "meshstack_workspace_tag" "first" {
  metadata = {
    # This resource cannot create the workspace it tags, so a missing one must be a clear error
    # rather than a nil-deref or a silent no-op. Applied without the workspace example's step.
    workspace_identifier = "does-not-exist-workspace"
    key                  = meshstack_tag_definition.first_tag.spec.key
  }
  spec = {
    values = ["12345"]
  }
}
