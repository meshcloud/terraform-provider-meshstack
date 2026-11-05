resource "meshstack_workspace_tag" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
    key                  = "billing-account"
  }
  spec = {
    values = ["12345"]
  }
}
