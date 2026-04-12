resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "cost-center-my-suffix"
    display_name = "Cost Center"

    value_type = {
      string = {}
    }
  }
}

