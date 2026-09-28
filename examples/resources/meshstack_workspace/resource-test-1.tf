resource "meshstack_workspace" "example" {
  metadata = {
    name = "test-ws-${var.suffix}"
    tags = {
      (meshstack_tag_definition.workspace_tag.spec.key) = ["12345"]
    }
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}
