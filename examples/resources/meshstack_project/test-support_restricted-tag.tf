# Used by TestAccProject/restricted_default_tag (resource-test-3.tf): on create the backend injects
# this default into every project, whether or not the caller declares the tag.

resource "meshstack_tag_definition" "restricted_tag" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-restricted-project-${var.suffix}"
    display_name = "Restricted Test Tag"
    restricted   = true

    value_type = {
      string = {
        default_value = "injected-default"
      }
    }
  }
}
