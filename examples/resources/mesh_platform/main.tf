# meshPlatform resource example

terraform {
  required_providers {
    meshstack = {
      source  = "meshcloud/meshstack"
      version = ">= 1.0"
    }
  }
}

provider "meshstack" {
  # Configuration via environment variables:
  # MESHSTACK_ENDPOINT
  # MESHSTACK_API_KEY  
  # MESHSTACK_API_SECRET
}

# Basic meshPlatform resource
resource "meshstack_mesh_platform" "example" {
  metadata = {
    name = "my-openstack-platform"
  }

  spec = {
    display_name   = "Production OpenStack Platform"
    platform_type  = "OpenStack"
    description    = "Production OpenStack platform for compute workloads"
    
    tags = {
      tier       = ["production"]
      capability = ["compute", "storage"]
      environment = ["production"]
      team        = ["platform-team"]
      region      = ["eu-west-1"]
    }
    
    config = {
      auth_url = "https://keystone.example.com:5000/v3"
      region   = "RegionOne"
    }
  }
}

# Another example with different platform type
resource "meshstack_mesh_platform" "azure_example" {
  metadata = {
    name = "my-azure-platform"
  }

  spec = {
    display_name   = "Staging Azure Platform"
    platform_type  = "Azure"
    description    = "Azure platform for development and staging workloads"
    
    tags = {
      environment = ["staging"]
      team        = ["dev-team"]
    }
    
    config = {
      tenant_id       = "12345678-1234-1234-1234-123456789abc"
      subscription_id = "87654321-4321-4321-4321-cba987654321"
      location        = "West Europe"
    }
  }
}

output "openstack_platform_id" {
  value = meshstack_mesh_platform.example.metadata.name
}

output "azure_platform_id" {
  value = meshstack_mesh_platform.azure_example.metadata.name
}