resource "meshstack_tag_definition" "single_select_numbers" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-ss-numbers-[[.suffix]]"
    display_name = "Test Single Select Numbers"
    value_type = {
      single_select = {
        options = [1, 2, 3]
      }
    }
  }
}
