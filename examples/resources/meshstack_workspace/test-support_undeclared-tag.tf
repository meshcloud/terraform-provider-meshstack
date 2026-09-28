# A second meshWorkspace tag definition the workspace does not declare. The backend still returns it
# as an empty-list entry, so the read path must reconcile it away instead of surfacing it as drift.

resource "meshstack_tag_definition" "undeclared_workspace_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-undeclared-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
