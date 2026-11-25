resource "meshstack_workspace" "test" {
  metadata = {
    name = "my-workspace"
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}
