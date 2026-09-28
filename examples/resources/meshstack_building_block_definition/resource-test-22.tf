# The gitlab definition with its pipeline trigger token rotated, which is what a write-only secret
# has to survive after an import.
# An example for gitlab_pipeline implementation with required attributes only
resource "meshstack_building_block_definition" "example_05_gitlab_pipeline" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Example Building Block"
    description  = var.description
  }

  version_spec = {
    draft = var.draft

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
        integration_ref = meshstack_integration.gitlab.ref
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
