# An example for github_workflows implementation with required attributes only
resource "meshstack_building_block_definition" "example_02_github_workflows" {
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
      workflow_ref = {
        display_name    = "Workflow Reference"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
      deploy_settings = {
        display_name    = "Deploy Settings"
        type            = "JSON_SCHEMA"
        assignment_type = "USER_INPUT"
        # Rendered as a form in meshPanel; the value reaches the block as JSON, like a CODE input.
        json_schema = jsonencode({
          type     = "object"
          required = ["region"]
          properties = {
            region   = { type = "string", enum = ["eu-central-1", "us-east-1"] }
            replicas = { type = "integer", minimum = 1 }
          }
        })
        display_order = 1
      }
    }

    deletion_mode = "PURGE"
    implementation = {
      github_workflows = {
        repository      = "example/building-block"
        branch          = "main"
        apply_workflow  = "apply.yml"
        integration_ref = { uuid = one(data.meshstack_integrations.all.integrations).metadata.uuid }
        # Optional flags, default false
        async                 = true
        omit_run_object_input = true
      }
    }

    outputs = {
      workflow_run_url = {
        display_name    = "Workflow Run URL"
        type            = "STRING"
        assignment_type = "RESOURCE_URL"
      }
    }
  }
}
