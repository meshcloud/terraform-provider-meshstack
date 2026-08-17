resource "meshstack_project" "example" {
  metadata = {
    name               = "test-proj-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }
  spec = {
    payment_method_identifier = meshstack_payment_method.example.metadata.name
    display_name              = "My Project's Display Name"
    tags = {
      (meshstack_tag_definition.project_tag.spec.key) = [
        "tag-value1",
        "tag-value2",
        "tag-valueN"
      ]
    }
  }
}
