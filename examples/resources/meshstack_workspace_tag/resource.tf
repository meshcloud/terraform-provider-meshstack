resource "meshstack_workspace_tag" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
    key                  = "cost-center"
  }
  spec = {
    values = ["12345"]
  }
}
