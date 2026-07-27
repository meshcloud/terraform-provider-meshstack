resource "meshstack_workspace_tags" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    tags = {
      "cost-center" = ["12345"]
    }
  }
}
