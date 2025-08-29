## v0.10.1

FIXES:
- Correctly handle external changes to `meshstack_project` tags.
- Prefer explicit provider configuration over environment variables.
- Fix issues with optional `value_type` attributes for `meshstack_tag_definition`.

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
