resource "meshstack_workspace" "test" {
  metadata = {
    name = "my-workspace"
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}

[[ example "data-sources/meshstack_workspace/data-source.tf" ]]
