data "meshstack_building_block" "example" {
  metadata = {
    uuid = "e2cc9cbb-cf1d-4dc0-8461-64140110b6dc" # Building block UUID
  }
}

# The computed ref drops straight into another building block's parent refs:
#
#   resource "meshstack_building_block" "child" {
#     spec = {
#       parent_building_block_refs = [data.meshstack_building_block.example.ref]
#       # ...
#     }
#   }
