resource "meshstack_building_block_definition" "example_tenant" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name        = "Test BB v3 Tenant Definition"
    description         = "A tenant-level building block definition for BB v3 resource tests"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = "my-platform-type" }]
  }

  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name           = "Name"
        description            = "Name of the resource"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      size = {
        display_name           = "Size"
        description            = "Size of the resource"
        type                   = "INTEGER"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        description            = "Target environment"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        selectable_values      = ["dev", "staging", "prod"]
      }
      # STRING-typed sensitive USER_INPUT: the consumer supplies it and the backend surfaces only its
      # sha256 hash in all_inputs. Passed to the terraform module below as a sensitive variable.
      api_key = {
        display_name           = "API Key"
        description            = "A sensitive API key handed to the terraform implementation"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        sensitive              = {}
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/tenant-building-block.git"
      }
    }

    outputs = {
      # The terraform module echoes the decrypted api_key back as this non-sensitive output. The
      # acceptance test asserts it equals the supplied secret, proving the runner decrypted the
      # sensitive input end to end.
      api_key_echo = {
        display_name    = "API Key Echo"
        type            = "STRING"
        assignment_type = "NONE"
      }
    }
  }
}
