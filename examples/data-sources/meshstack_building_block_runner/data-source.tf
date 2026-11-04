# Assign the ref to runner_ref in the version_spec of a building block definition.
data "meshstack_building_block_runner" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }
  spec = {
    display_name = "My Terraform Runner"
  }
}
