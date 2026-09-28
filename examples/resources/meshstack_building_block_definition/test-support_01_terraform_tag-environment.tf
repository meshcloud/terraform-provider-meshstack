resource "meshstack_tag_definition" "environment" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "environment-${var.tag_suffix}"
    display_name = "Environment"

    value_type = {
      multi_select = {
        options = ["dev", "prod"]
      }
    }
  }
}
