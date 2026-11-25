resource "meshstack_workspace_tags" "example" {
  workspace = data.meshstack_workspace.example.metadata.name
  tags = {
    "cost-center" = ["12345"]
  }
}
