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
    display_name = "Sensitive Input [[.suffix]]"
    description  = "Rejects making a sensitive input plain"
  }

  version_spec = {
    draft = true

    inputs = {
      CONNECTOR_PLAIN = {
        display_name    = "Connector Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("plaintext-secret-v1")
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
        ref_name          = "v1.0.0"
      }
    }
  }
}
