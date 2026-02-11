# An example for manual implementation with required attributes only
resource "meshstack_building_block_definition" "example_04_azure_devops_pipeline" {
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
        integration_ref = { uuid = "550e8400-e29b-41d4-a716-446655440000" }
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
