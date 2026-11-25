resource "meshstack_project_tag" "example" {
  project_ref = data.meshstack_project.example.ref
  key         = data.meshstack_tag_definition.cost_center.spec.key
  values      = ["12345"]
}
