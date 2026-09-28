resource "meshstack_workspace_tag" "second" {
  # depends_on serializes the two read-modify-write cycles. Without it Terraform applies them in
  # parallel and one tag is silently lost — the race documented in workspaceTagCaveats.
  depends_on = [meshstack_workspace_tag.first]

  metadata = {
    workspace_identifier = meshstack_workspace.example.metadata.name
    key                  = meshstack_tag_definition.second_tag.spec.key
  }
  spec = {
    values = ["second"]
  }
}
