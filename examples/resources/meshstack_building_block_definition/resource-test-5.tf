# An example for azure_devops_pipeline implementation with required attributes only
resource "meshstack_building_block_definition" "example_04_azure_devops_pipeline" {
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
        integration_ref = meshstack_integration.azuredevops.ref
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
