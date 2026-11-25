resource "meshstack_project_tags" "example" {
  project_ref = data.meshstack_project.example.ref
  tags = {
    (data.meshstack_tag_definition.cost_center.spec.key) = ["12345"]
  }
}
