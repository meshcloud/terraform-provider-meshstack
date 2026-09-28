# Prerequisite for the workspace steps that declare a tag: the definition their tag key references.
# Combined with test-support_variables.tf.

resource "meshstack_tag_definition" "workspace_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-workspace-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
