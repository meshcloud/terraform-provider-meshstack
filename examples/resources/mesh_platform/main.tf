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

# AWS meshPlatform resource example
resource "meshstack_mesh_platform" "aws_example" {
  metadata = {
    name = "my-aws-platform"
  }

  spec = {
    display_name   = "Production AWS Platform"
    platform_type  = "AWS"
    description    = "Production AWS platform for compute workloads"
    
    tags = {
      tier       = ["production"]
      capability = ["compute", "storage"]
      environment = ["production"]
      team        = ["platform-team"]
      region      = ["us-west-2"]
    }
    
    config = {
      aws = {
        account_id = "123456789012"
        region     = "us-west-2"
        role_arn   = "arn:aws:iam::123456789012:role/meshPlatformRole"
        external_id = "unique-external-id"
      }
    }
  }
}

# Basic meshPlatform resource (legacy example - keeping for compatibility)
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
    
    # Note: OpenStack config will be strongly typed in future updates
    # config = {
    #   auth_url = "https://keystone.example.com:5000/v3"
    #   region   = "RegionOne"
    # }
  }
}

# Another example with different platform type (legacy - keeping for compatibility)
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
    
    # Note: Azure config will be strongly typed in future updates
    # config = {
    #   tenant_id       = "12345678-1234-1234-1234-123456789abc"
    #   subscription_id = "87654321-4321-4321-4321-cba987654321"
    #   location        = "West Europe"
    # }
  }
}

output "aws_platform_id" {
  value = meshstack_mesh_platform.aws_example.metadata.name
}

output "openstack_platform_id" {
  value = meshstack_mesh_platform.example.metadata.name
}

output "azure_platform_id" {
  value = meshstack_mesh_platform.azure_example.metadata.name
}