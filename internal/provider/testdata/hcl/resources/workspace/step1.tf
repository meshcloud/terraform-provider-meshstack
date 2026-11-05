resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "cost-center"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}
