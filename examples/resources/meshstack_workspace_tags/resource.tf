resource "meshstack_workspace_tags" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    tags = {
      "budget-code" = ["12345"]
    }
  }
}
