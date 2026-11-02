data "meshstack_building_block_runner" "example" {
  metadata = {
    uuid = "98520496-627d-43e6-82da-ce499179ff3f"
  }
}

# The computed ref drops straight into a building block definition version:
#
#   version_spec = {
#     runner_ref = data.meshstack_building_block_runner.example.ref
#     # ...
#   }
