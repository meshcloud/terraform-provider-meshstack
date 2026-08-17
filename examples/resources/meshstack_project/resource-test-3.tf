resource "meshstack_project" "example" {
  # No attribute references the restricted tag definition, so without depends_on the backend could
  # still be missing it when the project is created — and inject nothing.
  depends_on = [meshstack_tag_definition.restricted_tag]

  metadata = {
    name               = "test-proj-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }
  spec = {
    payment_method_identifier = meshstack_payment_method.example.metadata.name
    display_name              = "My Project's Display Name"
    tags = {
      (meshstack_tag_definition.project_tag.spec.key) = ["blue"]
    }
  }
}
