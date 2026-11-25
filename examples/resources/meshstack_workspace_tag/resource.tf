resource "meshstack_workspace_tag" "example" {
  workspace = data.meshstack_workspace.example.metadata.name
  key       = "cost-center"
  values    = ["12345"]
}
