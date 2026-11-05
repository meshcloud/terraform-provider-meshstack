locals {
  tag_options = ["option_a", "option_b", "option_c"]
}

resource "meshstack_tag_definition" "single_select_local" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-ss-local-[[.suffix]]"
    display_name = "Test Single Select Local"
    value_type = {
      single_select = {
        options = local.tag_options
      }
    }
  }
}
