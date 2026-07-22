resource "meshstack_tag_definition" "restricted_tag" {
  spec = {
    # target_kind, key and the string default_value are set by the test (the key is randomized per
    # run). On create the backend injects this default into every resource of the target kind, whether
    # or not the caller declares the tag.
    target_kind = "meshProject"
    key         = "test-key"

    display_name = "Restricted Test Tag"
    restricted   = true

    value_type = {
      string = {
        default_value = "default"
      }
    }
  }
}
