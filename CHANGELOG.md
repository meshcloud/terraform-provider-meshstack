## Unreleased

FEATURES:

- New resource: `meshstack_location` for managing meshStack locations.
- Start implementing acceptance tests running against local meshStack

## v0.16.3

FIXES:

- Landing zone: unordered attributes are sets instead of lists.

## v0.16.2

FIXES:

- AWS SSO access token is required.

## v0.16.1

FIXES:

- Missing docs for integrations data source.

## v0.16.0

FEATURES:

- Restructured `meshstack_platform` authentication configuration for all platforms.
- Secrets now use nested `plaintext` field within credential objects.
- Integrations data source.

FIXES:

- Renamed fields: `user_look_up_strategy` → `user_lookup_strategy`, `service_account_config` → `service_account`.

## v0.15.0

FEATURES:

- Support multi select building block inputs.

FIXES:

- Restrict allowed value types for building block inputs, combined inputs and outputs.

## v0.14.0

FEATURES:

- Payment method resources.

## v0.13.0

FEATURES:

- Add metering config support to `meshstack_mesh_platform` resource and data source.
- Add quotas to `meshstack_landingzone` resource and data source.

FIXES:

- Correctly model nullable platform config fields.

## v0.12.4

FEATURES:

- Add quota definitions to meshPlatforms.

## v0.12.3

FIXES:

- Bump terraform-plugin-docs and fix docs.

## v0.12.2

FIXES:

- Fix possible nil-pointer issue when handling obfuscated secrets.

## v0.12.1

FIXES:

- Handle obfuscated secrets in meshPlatform Azure Type.

## v0.12.0

FEATURES:

- Added `meshstack_mesh_platform` resource.
- Added `meshstack_mesh_platform` data source.

FIXES:

- Fix landing zone data source.

## v0.11.0

FEATURES:

- Added `meshstack_mesh_landing_zone` resource.
- Added `meshstack_mesh_landing_zone` data source.
- Automatically set `type` inside platform_properties for landing zones.

FIXES:

- Fix landing zone status handling.
- Make `type` a read-only property for landing zones.

## v0.10.1

FIXES:

- Correctly handle external changes to `meshstack_project` tags.
- Prefer explicit provider configuration over environment variables.
- Fix issues with optional `value_type` attributes for `meshstack_tag_definition`.
- Add missing replication key field to `meshstack_tag_definition`.

## v0.10.0

FEATURES:

- Added `meshstack_workspace_user_binding` resource.
- Added `meshstack_workspace_group_binding` resource.

## v0.9.0

FEATURES:

- Added polling for building block (v2) and tenant (v4) resources until creation and deletion are complete.

## v0.8.0

FEATURES:

- Added `meshstack_workspace` resource.
- Added `meshstack_workspace` data source.
- Added `meshstack_tenant_v4` resource.
- Added `meshstack_tenant_v4` data source.

FIXES:

- Allow `value_code` in `meshstack_building_block_v2` and `meshstack_building_block` resources.
- Documentation.

## v0.7.1

FEATURES:

- Source provider configuration from environment variables.

## v0.7.0

FEATURES:

- Added `meshstack_building_block_v2` data source.
- Added `meshstack_building_block_v2` resource.

## v0.6.1

REFACTOR:

- Validate success responses by checking for HTTP status codes in the 2xx range

## v0.6.0

FEATURES:

- Added `meshstack_tag_definitions` data source.
- Added `meshstack_tag_definition` data source.
- Added `meshstack_tag_definition` resource.

## v0.5.5

FIXES:

- HTTP response code for tenant creation is now 201.
- HTTP response code for project creation is now 201.

## v0.5.4

FIXES:

- HTTP response code for building block creation is now 201.
