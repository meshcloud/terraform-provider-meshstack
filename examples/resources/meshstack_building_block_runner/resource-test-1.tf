resource "meshstack_building_block_runner" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My Terraform Runner"
    implementation_type = "TERRAFORM"
    public_key          = var.runner_public_key
    restriction         = "PRIVATE"
  }
}
