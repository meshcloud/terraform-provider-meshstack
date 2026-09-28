# The tag definition the workspace_tags map writes under. `variable "suffix"` and the workspace come
# from the meshstack_workspace example's step config (variant 3, the one without inline tags, so this
# resource owns the workspace's tags).

resource "meshstack_tag_definition" "first_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-wsts-first-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
