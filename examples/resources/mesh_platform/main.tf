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
    tags = {
      environment = ["production"]
      team        = ["platform-team"]
      region      = ["eu-west-1"]
    }
  }

  spec = {
    display_name   = "Production OpenStack Platform"
    platform_type  = "OpenStack"
    description    = "Production OpenStack platform for compute workloads"
    
    tags = {
      tier       = ["production"]
      capability = ["compute", "storage"]
    }
  }
}

# Another example with different platform type
resource "meshstack_mesh_platform" "azure_example" {
  metadata = {
    name = "my-azure-platform"
    tags = {
      environment = ["staging"]
    }
  }

  spec = {
    display_name   = "Staging Azure Platform"
    platform_type  = "Azure"
    description    = "Azure platform for development and staging workloads"
  }
}

output "openstack_platform_id" {
  value = meshstack_mesh_platform.example.metadata.name
}

output "azure_platform_id" {
  value = meshstack_mesh_platform.azure_example.metadata.name
}