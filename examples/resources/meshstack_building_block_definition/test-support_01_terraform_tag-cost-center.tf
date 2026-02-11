resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "cost-center-${random_string.tag_suffix.result}"
    display_name = "Cost Center"

    value_type = {
      string = {}
    }
  }
}

