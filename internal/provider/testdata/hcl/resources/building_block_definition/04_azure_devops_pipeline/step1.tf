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

resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"
        personal_access_token = {
          secret_value = "mock-pat-token-12345"
        }
      }
    }
  }
}

resource "meshstack_building_block_definition" "example_04_azure_devops_pipeline" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Example Building Block"
    description  = "An example building block definition"
  }

  version_spec = {
    draft = true

    inputs = {
      pipeline_config = {
        display_name    = "Pipeline Configuration"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    implementation = {
      azure_devops_pipeline = {
        project         = "MyProject"
        pipeline_id     = "42"
        integration_ref = meshstack_integration.example_azure_devops.ref
        ref_name        = "refs/heads/main"
      }
    }

    outputs = {
      pipeline_run_id = {
        display_name    = "Pipeline Run ID"
        type            = "STRING"
        assignment_type = "NONE"
      }
    }
  }
}
