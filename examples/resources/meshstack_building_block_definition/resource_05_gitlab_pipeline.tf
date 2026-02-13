# An example for gitlab_pipeline implementation with required attributes only
resource "meshstack_building_block_definition" "example_05_gitlab_pipeline" {
  metadata = {
    owned_by_workspace = "my-workspace"
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
          secret_value   = "glptt-..."
          secret_version = null
        }
        integration_ref = { uuid = "550e8400-e29b-41d4-a716-446655440000" }
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
