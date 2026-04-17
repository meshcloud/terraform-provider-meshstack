# An example for manual implementation with required attributes only
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
        integration_ref = { uuid = one(data.meshstack_integrations.all.integrations).metadata.uuid }
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
