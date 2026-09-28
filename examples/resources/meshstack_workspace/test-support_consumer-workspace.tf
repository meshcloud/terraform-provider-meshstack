# The consumer workspace of a cross-workspace test: it holds the restricted API key that reads
# meshObjects as a second workspace, so it must be distinct from meshstack_workspace.example.

resource "meshstack_workspace" "other" {
  metadata = {
    name = "test-ws-other-${var.suffix}"
    tags = {}
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}
