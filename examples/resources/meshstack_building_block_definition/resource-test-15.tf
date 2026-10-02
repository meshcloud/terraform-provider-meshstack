# An input named "plaintext" that is not a secret at all. The content hash disallows plaintext keys,
# and a key-name match used to reject this definition wrongly — hence the deliberate name.
# This example uses the Terraform implementation and defines all optional attributes
resource "meshstack_building_block_definition" "example_01_terraform" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
    tags = { # Optional
      (meshstack_tag_definition.environment.spec.key) = ["dev", "prod"]
      (meshstack_tag_definition.cost_center.spec.key) = ["cc-123"]
    }
  }

  spec = {
    display_name          = "Example Building Block"
    display_name_template = "Example Building Block {{ resource_name }}"
    # The test's own image: a generated config dir has no bb-symbol.png beside it, and the
    # provider resolves the path from its own working directory.
    symbol                    = provider::meshstack::load_image_file("testdata/images/image.png")
    description               = var.description
    readme                    = "# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes." # Optional
    support_url               = "https://support.example.com/building-blocks"                                                      # Optional
    documentation_url         = "https://docs.example.com/building-blocks"                                                         # Optional
    target_type               = "TENANT_LEVEL"                                                                                     # Optional: defaults to "WORKSPACE"
    supported_platforms       = [{ name = "AZURE" }, { name = "AWS" }]
    run_transparency          = true # Optional: defaults to false
    use_in_landing_zones_only = true # Optional: defaults to false
    # Only the email subscriber: a user: subscriber has to name a user that exists.
    notification_subscribers = ["email:ops@example.com"]

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
    draft = var.draft

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
      plaintext = { display_name = "Plaintext", type = "STRING", assignment_type = "STATIC", argument = jsonencode("hello") }
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
        pre_run_script = var.pre_run_script != null ? var.pre_run_script : "echo \"hello world\""
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
    dependency_refs = [meshstack_building_block_definition.other.ref]
  }
}
