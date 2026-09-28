resource "meshstack_tag_definition" "example" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-project-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
