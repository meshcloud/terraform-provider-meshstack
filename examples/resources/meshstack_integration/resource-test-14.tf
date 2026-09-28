# resource-test-1.tf without any config variant. The schema has to reject it at plan time; an empty
# config used to crash the provider at apply time.

resource "meshstack_integration" "example_github" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitHub Integration"
    config       = {}
  }
}
