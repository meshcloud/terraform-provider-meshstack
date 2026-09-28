# The tag definition the second meshstack_workspace_tag writes under, for the case that puts two of
# them on one workspace.

resource "meshstack_tag_definition" "second_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-wst-second-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
