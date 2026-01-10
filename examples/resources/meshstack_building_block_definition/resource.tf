resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
    tags = { # Optional
      environment = ["production", "staging"]
      team        = ["platform-team"]
      cost-center = ["cc-123"]
    }
  }

  spec = {
    display_name                      = "Example Building Block"
    symbol                            = "🏗️" # Optional
    description                       = "An example building block definition"
    readme                            = "# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes." # Optional
    support_url                       = "https://support.example.com/building-blocks"                                                      # Optional
    documentation_url                 = "https://docs.example.com/building-blocks"                                                         # Optional
    target_type                       = "TENANT"                                                                                           # Optional: defaults to "WORKSPACE"
    supported_platforms               = ["azure.platform", "aws.platform"]
    run_transparency                  = true                                     # Optional: defaults to false
    use_in_landing_zones_only         = true                                     # Optional: defaults to false
    notification_subscriber_usernames = ["admin@example.com", "ops@example.com"] # Optional
  }

  draft                      = true
  only_apply_once_per_tenant = false    # Optional: defaults to false
  deletion_mode              = "DELETE" # Optional: defaults to "DELETE"
  runner_ref                 = "my-runner"

  # Optional: Dependencies on other building blocks
  dependency_refs = [
    "dep-1",
    "dep-2"
  ]

  # Optional: Inputs for the building block
  inputs = {
    environment = {
      display_name           = "Environment"
      type                   = "SINGLE_SELECT"
      assignment_type        = "USER_INPUT"
      is_environment         = false                      # Optional: defaults to false
      is_sensitive           = false                      # Optional: defaults to false
      updateable_by_consumer = true                       # Optional: defaults to false
      selectable_values      = ["dev", "staging", "prod"] # Optional
      description            = "The target environment"   # Optional
    }
    resource_name = {
      display_name                   = "Resource Name"
      type                           = "STRING"
      assignment_type                = "USER_INPUT"
      is_environment                 = false                                                                     # Optional: defaults to false
      is_sensitive                   = false                                                                     # Optional: defaults to false
      updateable_by_consumer         = true                                                                      # Optional: defaults to false
      description                    = "Name of the resource to create"                                          # Optional
      value_validation_regex         = "^[a-z0-9-]+$"                                                            # Optional
      validation_regex_error_message = "Resource name must contain only lowercase letters, numbers, and hyphens" # Optional
    }
  }

  # Optional: Outputs from the building block
  outputs = {
    tenant_id = {
      display_name    = "Tenant ID"
      type            = "STRING"
      assignment_type = "PLATFORM_TENANT_ID"
    }
    sign_in_url = {
      display_name    = "Sign-in URL"
      type            = "STRING"
      assignment_type = "SIGN_IN_URL"
    }
  }

  # Optional: Implementation - Terraform or GitHub Actions
  implementation = {
    terraform = {
      terraform_version              = "1.9.0"
      repository_url                 = "https://github.com/example/building-block.git"
      async                          = false                       # Optional: defaults to false
      repository_path                = "terraform/modules/example" # Optional
      ref_name                       = "v1.0.0"                    # Optional - git ref (branch, tag, commit)
      use_mesh_http_backend_fallback = false                       # Optional: defaults to false

      # Optional: SSH configuration for private repositories
      ssh_private_key         = "-----BEGIN OPENSSH PRIVATE KEY-----\n..." # Optional: write-only, not stored in state
      ssh_private_key_version = "v1"                                       # Required when ssh_private_key is set

      # Optional: SSH known host configuration
      ssh_known_host = { # Optional
        host      = "github.com"
        key_type  = "ssh-rsa"
        key_value = "AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+..."
      }
    }

    # OR use GitHub Actions implementation
    # github_actions = {
    #   repository                      = "meshcloud/some-repo"
    #   branch                          = "main"
    #   apply_workflow                  = "apply.yml"
    #   destroy_workflow                = "destroy.yml" # optional
    #   source_platform_full_identifier = "my-platform.tenant-id"
    # }
  }
}
