resource "meshstack_tag_definition" "example" {
  spec = {
    target_kind = "meshProject"
    key         = "example-key"

    display_name = "Example"

    value_type = {
      email = {
        default_value    = "default"
        validation_regex = ".*"
      }
    }
    description = "Example Description"
    sort_order  = 0
    mandatory   = false
    immutable   = false
    restricted  = false
  }
}
