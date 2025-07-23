resource "meshstack_workspace" "example" {
  metadata = {
    name = "my-workspace-identifier"
    tags = {
      "cost-center" = [
        "12345"
      ]
    }
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}
