resource "meshstack_tag_definition" "tag" {
  spec = {
    # target_kind and key are set by the test (the key is randomized per run).
    target_kind = "meshProject"
    key         = "test-key"

    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
