data "meshstack_tag_definition" "example" {
  # A tag definition's identifier is <target_kind>.<key>, built here from the definition the test
  # creates so the lookup names something that exists.
  name = "${meshstack_tag_definition.example.spec.target_kind}.${meshstack_tag_definition.example.spec.key}"
}
