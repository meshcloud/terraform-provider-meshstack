resource "buildingblock" "example" {
  definition_uuid    = "some-definition-uuid"
  definition_version = 1
  tenant_identifier  = "tenantIdentifier"

  name         = "some-name"
  display_name = "displayname"

  inputs = {
    key_1 = "value"
    key_2 = 5
    key_3 = true
  }

  parents = [
    {
      definition_uuid = "some-parent-definition-uuid"
      uuid            = "some-parent-block-uuid"
    }
  ]
}
