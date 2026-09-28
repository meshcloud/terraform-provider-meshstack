# The second tag definition, for the case that drops one key of two from the map.

resource "meshstack_tag_definition" "second_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-wsts-second-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
