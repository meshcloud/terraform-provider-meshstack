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

resource "meshstack_integration" "example_gitlab" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitLab Integration"
    config = {
      gitlab = {
        base_url = "https://gitlab.com"
      }
    }
  }
}

# An example for gitlab_pipeline implementation with required attributes only
resource "meshstack_building_block_definition" "example_05_gitlab_pipeline" {
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
      deployment_env = {
        display_name    = "Deployment Environment"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    implementation = {
      gitlab_pipeline = {
        project_id = "12345678"
        ref_name   = "main"
        pipeline_trigger_token = {
          secret_value   = "updated-plaintext-secret"
          secret_version = "v1"
        }
        integration_ref = meshstack_integration.example_gitlab.ref
      }
    }

    outputs = {
      pipeline_web_url = {
        display_name    = "Pipeline URL"
        type            = "STRING"
        assignment_type = "RESOURCE_URL"
      }
    }
  }
}
