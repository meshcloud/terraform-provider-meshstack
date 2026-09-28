resource "meshstack_workspace" "example" {
  metadata = {
    name = "test-ws-${var.suffix}"
    tags = {}
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}
