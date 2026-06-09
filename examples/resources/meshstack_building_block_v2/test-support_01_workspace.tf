resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Test BB v2 Definition"
    description  = "A building block definition for BB v2 resource tests"
  }

  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name    = "Name"
        description     = "Name of the resource"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
      size = {
        display_name           = "Size"
        description            = "Size of the resource"
        type                   = "INTEGER"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name      = "Environment"
        description       = "Target environment"
        type              = "SINGLE_SELECT"
        assignment_type   = "USER_INPUT"
        selectable_values = ["dev", "staging", "prod"]
      }
      region = {
        display_name    = "Region"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("eu-west-1")
      }
      enable_monitoring = {
        display_name    = "Enable Monitoring"
        type            = "BOOLEAN"
        assignment_type = "STATIC"
        argument        = jsonencode(true)
      }
    }

    implementation = {
      manual = {}
    }

    # Outputs are omitted for manual building blocks: the backend derives them from the inputs, so
    # version_spec.outputs is computed and must not be set here.
  }
}
