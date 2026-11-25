resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_building_block_definition" "test" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Default runner secret [[.suffix]]"
    description  = "A definition with a sensitive input and without runner_ref"
  }

  version_spec = {
    draft = true

    inputs = {
      CONNECTOR_SECRET = {
        display_name    = "Connector Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value   = "plaintext-secret-v1"
            secret_version = "v1"
          }
        }
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
      }
    }
  }
}
