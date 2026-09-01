# This example uses the Terraform implementation and defines all optional attributes
resource "meshstack_building_block_definition" "example_01_terraform" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    tags = { # Optional
      "environment" = ["dev", "prod"]
      "cost-center" = ["cc-123"]
    }
  }

  spec = {
    display_name              = "Example Building Block"
    display_name_template     = "Example Building Block {{ resource_name }}"                         # Optional: names each ordered building block after its inputs
    symbol                    = provider::meshstack::load_image_file("${path.module}/bb-symbol.png") # Optional
    description               = "An example building block definition"
    readme                    = "# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes." # Optional
    support_url               = "https://support.example.com/building-blocks"                                                      # Optional
    documentation_url         = "https://docs.example.com/building-blocks"                                                         # Optional
    target_type               = "TENANT_LEVEL"                                                                                     # Optional: defaults to "WORKSPACE"
    supported_platforms       = [{ name = "AZURE" }, { name = "AWS" }]
    run_transparency          = true                                            # Optional: defaults to false
    use_in_landing_zones_only = true                                            # Optional: defaults to false
    notification_subscribers  = ["user:some-username", "email:ops@example.com"] # Optional, note user: and email: prefix

    # Optional: which run triggers need an operator's approval before the run is applied.
    # Defaults to no approval gate at all. Only the terraform implementation supports approval policies, because an
    # approver reviews the planned changes of a dry run. Flags left out default to false.
    approval_policies = {
      version_upgrade = true
      manual_triggers = true
    }

    # Optional: drift detection / reconciliation schedule. Defaults to mode = "DISABLED".
    # DRIFT_DETECTION only reports drift and needs the terraform implementation; DRIFT_RECONCILIATION
    # also fixes it and works with every implementation except manual.
    schedule = {
      mode      = "DRIFT_DETECTION"
      frequency = "DAILY"
    }
  }

  version_spec = {
    draft = true

    # Optional: Specify runner if necessary (otherwise, shared runner is used)
    runner_ref = {
      kind = "meshBuildingBlockRunner"
      uuid = "98520496-627d-43e6-82da-ce499179ff3f"
    }

    only_apply_once_per_tenant = true     # Optional: defaults to false
    deletion_mode              = "DELETE" # Optional: defaults to "DELETE"

    # Optional: API permissions provided to building block runs via an ephemeral API key
    permissions = ["TENANT_LIST", "TENANT_SAVE"]

    # Optional: Inputs for the building block
    inputs = {
      environment = {
        display_name      = "Environment"
        description       = "The target environment" # Optional
        type              = "SINGLE_SELECT"
        assignment_type   = "USER_INPUT"
        selectable_values = ["dev", "prod", "staging"] # Optional, must be non-empty
        is_optional       = true                       # Optional: defaults to false
        display_order     = 1
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
        display_order                  = 2                                                                         # Optional: arranges inputs in meshPanel; part of the content hash, so it cannot change on a released version
      }
      deploy_settings = {
        display_name    = "Deploy Settings"
        type            = "JSON_SCHEMA"
        assignment_type = "USER_INPUT"
        # This input gets a form of its own: meshPanel renders it from the schema, and what it produces
        # reaches the building block as JSON text.
        json_schema = jsonencode({
          type     = "object"
          required = ["region"]
          properties = {
            region   = { type = "string", enum = ["eu-central-1", "us-east-1"] }
            replicas = { type = "integer", minimum = 1 }
          }
        })
        display_order = 3
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
      business_unit = {
        display_name    = "Business Unit"
        description     = "The business unit tag of the workspace this building block belongs to" # Optional
        type            = "CODE"                                                                  # Tag inputs are always CODE: a tag value is a list of strings
        assignment_type = "TAG"
        # Names the tag to read as "<target>.<tagKey>". A TENANT_LEVEL building block can read WORKSPACE,
        # PROJECT, PAYMENT_METHOD and LANDING_ZONE tags; a WORKSPACE_LEVEL one only WORKSPACE tags.
        argument      = jsonencode("WORKSPACE.${meshstack_tag_definition.workspace_business_unit.spec.key}")
        display_order = 4
      }
      "some-file.yaml" = {
        display_name    = "Some input file"
        type            = "FILE"
        assignment_type = "STATIC"
        argument        = jsonencode(provider::meshstack::load_file("${path.module}/some-file.yaml"))
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

        # Optional: Shell script executed after 'tofu init' and before 'tofu apply'/'tofu destroy'.
        pre_run_script = "echo \"hello world\""
      }
    }

    # Optional: Outputs from the building block
    outputs = {
      some_output_flag = {
        display_name    = "If true, it really worked"
        type            = "BOOLEAN"
        assignment_type = "NONE"
        display_order   = 1
      }
      summary = {
        display_name    = "Summary of work"
        type            = "STRING"
        assignment_type = "SUMMARY"
        display_order   = 2
      }
    }

    # Optional: Dependencies on other building blocks, prefer using .ref output attributes.
    dependency_refs = [one(data.meshstack_building_block_definitions.example.building_block_definitions).ref]
  }
}
