data "meshstack_building_block_runner" "example" {
  spec = {
    display_name = "My Terraform Runner"
  }
  # Narrow the lookup when several runners share that name, or look it up by UUID:
  # metadata = {
  #   owned_by_workspace = "my-workspace"
  #   uuid               = "6a1f4e9c-2b7d-4c3a-9e8f-1d5b0c7a3e42"
  # }
}

# Without metadata and spec, it reads the runner meshStack hosts:
#
#   data "meshstack_building_block_runner" "hosted" {}

# The computed ref drops straight into a building block definition version:
#
#   version_spec = {
#     runner_ref = data.meshstack_building_block_runner.example.ref
#     # ...
#   }
