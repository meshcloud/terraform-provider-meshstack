resource "meshstack_tag_definition" "environment" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "environment-${random_string.tag_suffix.result}"
    display_name = "Environment"

    value_type = {
      multi_select = {
        options = ["dev", "prod"]
      }
    }
  }
}
