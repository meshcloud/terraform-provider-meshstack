resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

# This example uses the Terraform implementation and defines all optional attributes
resource "meshstack_building_block_definition" "example_01_terraform" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    tags               = { (meshstack_tag_definition.environment.spec.key) = ["dev", "prod"], (meshstack_tag_definition.cost_center.spec.key) = ["cc-123"] }
  }

  spec = {
    display_name              = "Example Building Block"
    display_name_template     = "Example Building Block {{ resource_name }}"                      # Optional: names each ordered building block after its inputs
    symbol                    = provider::meshstack::load_image_file("testdata/images/image.png") # Optional
    description               = "An example building block definition"
    readme                    = "# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes." # Optional
    support_url               = "https://support.example.com/building-blocks"                                                      # Optional
    documentation_url         = "https://docs.example.com/building-blocks"                                                         # Optional
    target_type               = "TENANT_LEVEL"                                                                                     # Optional: defaults to "WORKSPACE"
    supported_platforms       = [{ name = "AZURE" }, { name = "AWS" }]
    run_transparency          = true                      # Optional: defaults to false
    use_in_landing_zones_only = true                      # Optional: defaults to false
    notification_subscribers  = ["email:ops@example.com"] # Optional, note user: and email: prefix

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
    runner_ref = meshstack_building_block_runner.example_with_wif.ref

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
        type            = "JSON"
        assignment_type = "USER_INPUT"
        condition       = "input.environment == 'prod'"
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
        is_optional     = true # Optional: run without the value while the tag is unset, instead of waiting for it
        # Names the tag to read as "<target>.<tagKey>". A TENANT_LEVEL building block can read WORKSPACE,
        # PROJECT, PAYMENT_METHOD and LANDING_ZONE tags; a WORKSPACE_LEVEL one only WORKSPACE tags.
        argument      = jsonencode("WORKSPACE.${meshstack_tag_definition.workspace_business_unit.spec.key}")
        display_order = 4
      }
      "some-file.yaml" = {
        display_name    = "Some input file"
        type            = "FILE"
        assignment_type = "STATIC"
        argument        = jsonencode(provider::meshstack::encode_file("some-content"))
      }
    }

    implementation = {
      terraform = {
        terraform_version              = "1.9.0"
        repository_url                 = "https://github.com/example/building-block.git"
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
    dependency_refs = [meshstack_building_block_definition.other.ref]
  }
}

resource "meshstack_tag_definition" "environment" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "environment-[[.suffix]]"
    display_name = "Environment"

    value_type = {
      multi_select = {
        options = ["dev", "prod"]
      }
    }
  }
}

resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"

    value_type = {
      string = {}
    }
  }
}

# The workspace tag that the building block definition reads as a TAG input.
resource "meshstack_tag_definition" "workspace_business_unit" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "business-unit-[[.suffix]]"
    display_name = "Business Unit"

    value_type = {
      string = {}
    }
  }
}

# A minimal manual BBD used as a dependency for the terraform example
resource "meshstack_building_block_definition" "other" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Dependency Building Block"
    description  = "A building block that is used as a dependency"
  }

  version_spec = {
    draft = false
    implementation = {
      manual = {}
    }
  }
}

resource "meshstack_building_block_runner" "example_with_wif" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuP1aaqG5oCtnyPUWVbor\n11jsYaOt5A/IzOCA8QsrKFIpMlPOTl9HNsKDDUmYPMNI6uNCpFHmcV46/cqr47s8\ns8NwxckJeHLDItGXGJ8Z2k2YUSp4UPH0PxTNqr5XDIzucxYEoRPluy3jin54B6qU\ngd/BSLJq5e9Ifx00G24FMqJ1Bc24/lYVnBxuXkFHGXJMpfVTgOPlEbIjHYeSfB6U\nQVjUsLddpJCKG/dtQYLrufH3k7A8d6G3PlMe+BNOJpNc9zSxzd6bgbY50Q8BHnmI\nD6/hNv0bNwwerI9eVDrsyDzQ/5YiK+APjAHrzqpCizooYyHgzvqN6aHQEGm4EC0w\nVwIDAQAB\n-----END PUBLIC KEY-----\n"
    restriction         = "PRIVATE"

    workload_identity_federation = {
      # meshStack fills in the placeholders for each building block definition and reports the result as
      # that definition's version_latest.workload_identity_federation.subject.
      subject_template = "system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"
      issuer           = "https://oidc.example.com"

      gcp = {
        audience   = "gcp-workload-identity-provider:namespace"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}

resource "meshstack_building_block_runner" "example_with_other_wif" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My Other GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuP1aaqG5oCtnyPUWVbor\n11jsYaOt5A/IzOCA8QsrKFIpMlPOTl9HNsKDDUmYPMNI6uNCpFHmcV46/cqr47s8\ns8NwxckJeHLDItGXGJ8Z2k2YUSp4UPH0PxTNqr5XDIzucxYEoRPluy3jin54B6qU\ngd/BSLJq5e9Ifx00G24FMqJ1Bc24/lYVnBxuXkFHGXJMpfVTgOPlEbIjHYeSfB6U\nQVjUsLddpJCKG/dtQYLrufH3k7A8d6G3PlMe+BNOJpNc9zSxzd6bgbY50Q8BHnmI\nD6/hNv0bNwwerI9eVDrsyDzQ/5YiK+APjAHrzqpCizooYyHgzvqN6aHQEGm4EC0w\nVwIDAQAB\n-----END PUBLIC KEY-----\n"
    restriction         = "PRIVATE"

    workload_identity_federation = {
      # meshStack fills in the placeholders for each building block definition and reports the result as
      # that definition's version_latest.workload_identity_federation.subject.
      subject_template = "system:serviceaccount:other-namespace:bbd.{{ buildingBlockDefinitionUuid }}"
      issuer           = "https://oidc-other.example.com"

      gcp = {
        audience   = "gcp-workload-identity-provider:namespace"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}

data "meshstack_building_block_definitions" "example" {
  # Optional server-side filter: limit to one workspace.
  # Additional filtering (e.g. by display name) is done in Terraform expressions.
  workspace_identifier = meshstack_building_block_definition.example_01_terraform.metadata.owned_by_workspace
  uuids                = [meshstack_building_block_definition.example_01_terraform.metadata.uuid]
}

# Example: select exactly one BBD by display name pattern.
# (Use can(regex(...)) so non-matching entries are filtered without evaluation errors.)
locals {
  selected_bbd = one([
    for bbd in data.meshstack_building_block_definitions.example.building_block_definitions : bbd
    if can(regex("^Example Building Block.*$", bbd.spec.display_name))
  ])
}
