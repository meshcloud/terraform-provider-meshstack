resource "meshstack_platform" "example" {
  metadata = {
    name = "my-platform-identifier"
  }

  spec = {
    display_name  = "My Cloud Platform"
    platform_type = "azure"
    
    config = {
      subscription_id = "12345678-1234-1234-1234-123456789012"
      tenant_id      = "87654321-4321-4321-4321-210987654321"
      location       = "West Europe"
    }

    tags = {
      environment = ["production"]
      team        = ["platform", "ops"]
      cost_center = ["cc-12345"]
    }
  }
}