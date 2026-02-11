# This example uses the Terraform implementation and defines all optional attributes
resource "meshstack_building_block_definition" "example_01_terraform" {
  metadata = {
    owned_by_workspace = "my-workspace"
    tags = { # Optional
      "environment" = ["dev", "prod"]
      "cost-center" = ["cc-123"]
    }
  }

  spec = {
    display_name              = "Example Building Block"
    symbol                    = "🏗️" # Optional
    description               = "An example building block definition"
    readme                    = "# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes." # Optional
    support_url               = "https://support.example.com/building-blocks"                                                      # Optional
    documentation_url         = "https://docs.example.com/building-blocks"                                                         # Optional
    target_type               = "TENANT_LEVEL"                                                                                     # Optional: defaults to "WORKSPACE"
    supported_platforms       = [{ name = "AZURE" }, { name = "AWS" }]
    run_transparency          = true                                            # Optional: defaults to false
    use_in_landing_zones_only = true                                            # Optional: defaults to false
    notification_subscribers  = ["user:some-username", "email:ops@example.com"] # Optional, note user: and email: prefix
  }

  version_spec = {
    draft = true

    # Optional: Specify runner if necessary (otherwise, shared runner is used)
    runner_ref = {
      kind = "meshBuildingBlockRunner"
      uuid = "66ddc814-1e69-4dad-b5f1-3a5bce51c01f"
    }

    only_apply_once_per_tenant = false    # Optional: defaults to false
    deletion_mode              = "DELETE" # Optional: defaults to "DELETE"

    # Optional: Inputs for the building block
    inputs = {
      environment = {
        display_name      = "Environment"
        description       = "The target environment" # Optional
        type              = "SINGLE_SELECT"
        assignment_type   = "USER_INPUT"
        selectable_values = ["dev", "prod", "staging"] # Optional
      }
      resource_name = {
        display_name                   = "Resource Name"
        description                    = "Name of the resource to create" # Optional
        type                           = "STRING"
        assignment_type                = "USER_INPUT"
        default_value                  = jsonencode("some-resource-name")
        updateable_by_consumer         = true                                                                      # Optional: defaults to false
        value_validation_regex         = "^[a-z0-9-]+$"                                                            # Optional
        validation_regex_error_message = "Resource name must contain only lowercase letters, numbers, and hyphens" # Optional
      }
      SOMETHING_VERY_SECRET = {
        display_name    = "Top Secret"
        description     = "Really secret" # Optional
        type            = "STRING"
        assignment_type = "STATIC"
        is_environment  = true # Optional: defaults to false
        sensitive = {
          argument = {
            secret_value = "write-only-plaintext-value-should-be-ephemeral"
          }
        }
      }
    }

    implementation = {
      terraform = {
        terraform_version              = "1.9.0"
        repository_url                 = "https://github.com/example/building-block.git"
        async                          = true                        # Optional: defaults to false
        repository_path                = "terraform/modules/example" # Optional
        ref_name                       = "v1.0.0"                    # Optional - git ref (branch, tag, commit)
        use_mesh_http_backend_fallback = true                        # Optional: defaults to false

        # Optional: SSH configuration for private repositories
        ssh_private_key = {
          secret_value   = "-----BEGIN OPENSSH PRIVATE KEY-----\n..." # write-only, not stored in state
          secret_version = null                                       # change whenever value shall be re-applied
        }

        # Optional: SSH known host configuration
        ssh_known_host = { # Optional
          host      = "github.com"
          key_type  = "ssh-rsa"
          key_value = "AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+..."
        }
      }
    }

    # Optional: Outputs from the building block
    outputs = {
      some_output_flag = {
        display_name    = "If true, it really worked"
        type            = "BOOLEAN"
        assignment_type = "NONE"
      }
      summary = {
        display_name    = "Summary of work"
        type            = "STRING"
        assignment_type = "SUMMARY"
      }
    }

    # Optional: Dependencies on other building blocks, prefer using using the .ref output attribute instead of hardcoding UUIDs
    dependency_refs = [{ uuid = "d161e3bf-c3e7-45f2-aa21-28de14593a74" }]
  }
}
