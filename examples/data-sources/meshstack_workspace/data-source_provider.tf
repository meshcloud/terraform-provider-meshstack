# Omit metadata.name to read the workspace that the provider acts in.
provider "meshstack" {
  workspace = "my-workspace"
}

data "meshstack_workspace" "this" {
}
