# Tenant-level building block definition for the v1->v3 migration acceptance test
# (04_tenant_moved_from_v1). It deliberately uses the manual implementation and no sensitive inputs:
# the migration starts from a legacy meshstack_buildingblock (v1) resource, whose input shape cannot
# carry sensitive values, so this fixture is kept separate from the terraform + sensitive-input
# showcase in test-support_02_tenant.tf / resource_02_tenant.tf.
resource "meshstack_building_block_definition" "example_tenant" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name        = "Test BB v3 Tenant Migration Definition"
    description         = "A tenant-level building block definition for the BB v3 v1->v3 migration test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = "my-platform-type" }]
  }

  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name           = "Name"
        description            = "Name of the resource"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      size = {
        display_name           = "Size"
        description            = "Size of the resource"
        type                   = "INTEGER"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        description            = "Target environment"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        selectable_values      = ["dev", "staging", "prod"]
      }
    }

    implementation = {
      manual = {}
    }
  }
}
