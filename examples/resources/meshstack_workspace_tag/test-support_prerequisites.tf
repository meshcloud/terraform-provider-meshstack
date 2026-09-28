# The tag definition the first meshstack_workspace_tag writes under. `variable "suffix"` and the
# workspace itself come from the meshstack_workspace example's step config (variant 3, the one
# without inline tags, so these resources own the workspace's tags).

resource "meshstack_tag_definition" "first_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-wst-first-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
