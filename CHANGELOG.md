# v0.24.1

BREAKING CHANGES:
- Release binaries are now published only for `linux` and `darwin` on `amd64` and `arm64`. Builds for `windows`, `freebsd`, `386` and 32-bit `arm` are no longer produced.
- `meshstack_tenant` and `meshstack_tenants` now use the meshTenant v4 GA media type instead of the v4 preview media type. They require a meshStack backend that has promoted meshTenant v4 to GA; older backends that only serve the `-preview` media type return HTTP 415 (Unsupported Media Type).
- The deprecated `meshstack_tenant_v4` resource and data source have been removed. Migrate to `meshstack_tenant` / `meshstack_tenants`. Because v0.24.1 no longer knows the `meshstack_tenant_v4` type, add a `moved` block on v0.24.0 (which still ships both the `moved` support and the deprecated type) and apply it before upgrading to v0.24.1.

# v0.24.0

BREAKING CHANGES:
- `meshstack_tenant` now runs on the meshTenant v4 API and the resource body changed accordingly (existing state is upgraded in place, without recreating the tenant): `metadata.platform_identifier` → `spec.platform_ref` (`{uuid, kind}`), `spec.local_id` → `spec.platform_tenant_id`, `spec.landing_zone_identifier` → `spec.landing_zone_ref` (`{name, kind}`), `metadata.assigned_tags` → `status.tags`, and `spec.quotas` is now a set. New computed `ref`, richer `status`, and a `wait_for_completion` toggle. Import accepts either the tenant UUID or the legacy `workspace.project.platform.location` composite. Requires meshStack with the meshTenant v4 API carrying `platformRef`/`landingZoneRef`. Upgrade caveats: the automatic state upgrade queries the backend and expects exactly one active tenant for the composite (it errors otherwise); and if you previously omitted the landing zone, set `landing_zone_ref` after upgrading, because it is `RequiresReplace` — leaving it unset forces a tenant recreate.
- `meshstack_tenants` (data source) now runs on the ref-based meshTenant v4 body: each returned tenant exposes `spec.platform_ref` (`{uuid, kind}`) and `spec.landing_zone_ref` (`{name, kind}`) instead of `spec.platform_identifier` / `spec.landing_zone_identifier`, plus a computed `ref` and a richer `status`. Update any consumers that read the old identifier fields.
- `meshstack_tenant` and `meshstack_tenants` no longer expose the lifecycle status outputs `metadata.created_on`, `metadata.marked_for_deletion_on` and `metadata.deleted_on`. These were read-only passthrough attributes with no functional use; lifecycle status is intentionally hidden inside the resource/data source (aligned with modern resources like `meshstack_building_block_definition`). Remove any references to these attributes.

FEATURES:
- `meshstack_landingzone` resource and data source now expose a computed `ref` output (`{name, kind}`) suitable for use as `landing_zone_ref` in tenant resources, matching the existing `ref` outputs on other resources.
- `meshstack_tenant` supports migrating from the deprecated `meshstack_tenant_v4` with a `moved` block. The move carries over the tenant uuid; the post-move refresh re-reads the tenant from the API to translate the v4 `spec.platform_identifier` into the ref-based `spec.platform_ref`, so the move does not recreate the tenant.
- `meshstack_location` and `meshstack_platform_type` `ref` outputs now include a computed `kind`, consistent with every other meshObject reference. The remaining hand-rolled `ref` / `*_ref` schemas (`meshstack_platform` `location_ref`, custom-platform `platform_type_ref`) are now defined through the shared meshRef helper.

FIXES:
- `meshstack_tenant`: changing only `wait_for_completion` (a client-side toggle with no API call) is now applied in place instead of failing with "Tenants can't be updated". Any other change to an existing tenant remains unsupported.

DEPRECATIONS:
- `meshstack_tenant_v4` (resource and data source) is deprecated in favor of `meshstack_tenant` / `meshstack_tenants`; migrate the resource with a `moved` block. It stays registered for now and is removed in a later release.

# v0.23.3

FEATURES:
- `meshstack_building_block_definition`: `version_spec.inputs` now support the `MESHSTACK_TENANT_UUID` assignment type, which assigns the meshTenant's UUID as a string. Like `PLATFORM_TENANT_ID`, it takes no `argument`, `default_value`, or `sensitive` value.
- `meshstack_building_block_definition`: manual building blocks (`implementation.manual`) now accept any special output `assignment_type` — `SIGN_IN_URL`, `RESOURCE_URL`, and `SUMMARY` in addition to `PLATFORM_TENANT_ID` — to mark how a derived output is used. Previously only `PLATFORM_TENANT_ID` was allowed. The output key must match an input key; the backend still derives the output set from the inputs.
- `meshstack_building_block` and `meshstack_building_block_v2`: `spec.building_block_definition_version_ref` now carries a `kind` field (computed, always `meshBuildingBlockDefinitionVersion`), consistent with every other meshObject reference. The `meshstack_building_block_definition` computed `version_latest`, `version_latest_release` and `versions` outputs, and the `meshstack_building_block_v2` / `meshstack_building_blocks` data sources, expose the matching `kind` too, so wiring a version output into a building block keeps working.

FIXES:
- `meshstack_building_block_definition`: `display_order` on `version_spec.inputs` and `version_spec.outputs` again counts towards a version's computed `content_hash`. Changing only `display_order` on a draft does not really need to rerun the building block, but hashing it keeps the backend simpler and the provider follows the same rule. The `content_hash` version was raised from 2 to 3 to go with this change. A `content_hash` written by an older provider is now recomputed at the current version instead of being treated as changed, so upgrading no longer shows false plan diffs or reruns building blocks whose definition is already released.
- `meshstack_platform` and `meshstack_landingzone`: `project_role_ref` in platform role mappings now has an optional `kind` (defaulting to `meshProjectRole`) instead of a read-only one, and no longer requires the identifier to be a known literal, so assigning a referenced resource's computed `ref` no longer fails with "Cannot set value for this attribute as the provider has marked it as read-only". This brings these references in line with the handling other meshObject references already have.
- `meshstack_platform`: `aws` (both `aws_sso` and `aws_identity_store`) and `gcp` role mappings are now modeled as unordered sets instead of ordered lists, matching how the backend stores them (keyed by the referenced meshProjectRole). Previously the backend was free to return these mappings in a different order than configured — for `aws` and `gcp` it round-trips them through a map — which surfaced as a permanent, no-op plan diff; as sets they are compared by value and no longer churn. The HCL block syntax is unchanged, so existing configurations keep working; on the first plan after upgrading you may see a one-time diff while Terraform re-tracks the elements.

# v0.23.2

FEATURES:
- `meshstack_building_block_definition`: `version_spec.inputs` and `version_spec.outputs` now support an optional `display_order` (number) attribute that controls how inputs/outputs are arranged in meshPanel. It defaults to the already set value when omitted, or `0` if there is none.

FIXES:
- `meshstack_building_block`: A `content_hash` wired from a definition's computed `content_hash` no longer triggers a re-run when the value changed only because the hash-algorithm version changed (e.g. after upgrading the provider). Such version-only differences are now recognized and ignored, while genuine content changes still re-run. Setting an arbitrary (non-versioned) `content_hash` to force a manual re-run continues to work.
- `meshstack_building_block` / `meshstack_building_block_v2`: Deleting a building block whose definition uses `deletion_mode = PURGE` no longer intermittently fails with "reached FAILED state during deletion" when the block's delete run itself fails. A purge removes the block regardless of that run's outcome, so a transient `FAILED` status is now tolerated and the provider waits for the lifecycle to reach `DELETED`. A failed deletion that is *not* being purged still surfaces as an error.

# v0.23.1

FIXES:
- `meshstack_building_block`: Prepare for the upcoming `WAITING_FOR_APPROVAL` run status (an approval gate coming soon to meshStack — not available yet). Once meshStack starts returning it, an awaited create/update where the run parks for approval will complete with a "waiting for input" warning instead of failing with "unknown building block status; provider may be out of date".
- `meshstack_building_block_definition`: `Read` now derives `version_spec.draft` from the definition's actual latest version instead of retaining the prior state value. Previously, when the latest version was switched to a draft outside Terraform, refresh kept `draft = false`, so a `draft = false -> true` change created a redundant new version instead of reconciling the existing draft in place.

# v0.23.0

FEATURES:
- New `meshstack_building_block` resource, the recommended way to manage building blocks.<br>
  Updates inputs, `spec.display_name` and the definition version in place instead of forcing a destroy and recreate ([#141](https://github.com/meshcloud/terraform-provider-meshstack/issues/141)).<br>
  Reruns the building block when a draft version's `content_hash` or a sensitive input's `secret_version` changes.<br>
  Manages only the inputs declared in `spec.inputs`. Inputs set by someone else stay untouched and show up read only in the computed `all_inputs`.<br>
  Each input takes one `jsonencode(...)` `value` (or a `sensitive` block) instead of the per-type `value_string`, `value_int` and similar attributes.<br>
  Optional `timeouts` (create, update, delete; default 30m) and `purge_on_delete`. Computed `status.latest_run_uuid` and `status.latest_dry_run_uuid`.<br>
  Migrate from the deprecated `meshstack_buildingblock` and `meshstack_building_block_v2` resources with a `moved` block. Sensitive inputs survive a `meshstack_building_block_v2` migration: re-declare the input with any placeholder `secret_value` to keep the current secret, and bump `secret_version` with a real value to rotate it.
- New `meshstack_building_blocks` data source to list building blocks, read only and aligned to the `meshstack_building_block` resource, with optional filters. Only active building blocks are returned.

FIXES:
- `meshstack_landingzone`: `spec.platform_ref` and `spec.mandatory_building_block_refs`/`spec.recommended_building_block_refs` now accept a referenced resource's computed `ref` directly (e.g. `platform_ref = meshstack_platform.example.ref`). The `kind` field is now optional (defaulting to its single valid value) instead of read-only, so assigning a full ref object no longer fails with "Cannot set value for this attribute as the provider has marked it as read-only".

# v0.22.1

FEATURES:
- `meshstack_integrations`: Added support for Entra ID SSO integrations via a new `spec.config.entraid` block (`tenant_id`, `client_id`, `client_secret`, and a computed `redirect_url`). Entra ID integrations can only be owned by the admin workspace, and can only be managed if your meshStack is configured to support Entra ID SSO integrations.

FIXES:
- `meshstack_building_block_definition`: The "version_spec cannot be updated in non-draft state" error now includes actionable next steps: set `draft = true` and apply to create a draft, then set `draft = false` and apply again to release.

# v0.22.0

Requires meshStack 2026.24.0 or later due to roll out of one shared meshcloud hosted building block runner.

BREAKING CHANGES:
- `meshstack_building_block_definition`: For manual building blocks, `version_spec.outputs` is now computed from the API and must be omitted from configuration — the backend derives one output per input. Configuring an output with any `assignment_type` other than `PLATFORM_TENANT_ID` is now rejected. Remove `outputs` blocks from manual building block definitions; you may still declare an output with `assignment_type = "PLATFORM_TENANT_ID"` to mark which output carries the tenant id.

FIXES:
- `meshstack_building_block_definition`: Fixed "Provider produced inconsistent result after apply" for manual building blocks whose outputs the backend auto-generates from inputs (e.g. `SINGLE_SELECT`/`STATIC` inputs), including when toggling `version_spec.draft` from `false` to `true` together with input changes ([#131](https://github.com/meshcloud/terraform-provider-meshstack/issues/131), [#176](https://github.com/meshcloud/terraform-provider-meshstack/issues/176)). Outputs are now reconciled from the API response.
- `meshstack_building_block_definition`: Rotating a sensitive input's secret on a released (immutable) version now fails with a clear "Updating a version_spec in non-draft state is not allowed" error instead of an opaque "Failed to determine content hash ... [plaintext]" error ([#196](https://github.com/meshcloud/terraform-provider-meshstack/issues/196)). Set `version_spec.draft = true` to create a new draft version; the secret rotation can be applied in the same step.
- `meshstack_building_block_definition`: Fixed an erroneous "Failed to determine content hash" error for definitions whose `version_spec.inputs`/`version_spec.outputs` contain an entry named `plaintext`. The secret safeguard now inspects the typed secret values instead of matching the literal `plaintext` JSON key, so user-chosen input/output names are no longer mistaken for secrets.
- When no `runner_ref` is provided, the new shared building block runner UUID `98520496-627d-43e6-82da-ce499179ff3f` is used which is suitable for all implementation types.
  Existing `building_block_definition` resources will see a plan change addressing this migration to a single shared runner. 
  Using the old shared runner UUIDs is deprecated but handled gracefully by the API.

# v0.21.0

Requires meshStack 2026.23.0 or later (previously 2026.22.0).

BREAKING CHANGES:
- `meshstack_building_block_v2`: The `spec.inputs` and `status.outputs` fields have changed from arrays to maps in the upstream API.
  The internal client representation has been updated accordingly. No Terraform schema changes are required, but this requires meshStack 2026.23.0 or later.
- `meshstack_building_block_v2`: The upstream API handles sensitive inputs as embedded secrets to align with the Building Block Definition API.
  Added `value_string_sensitive` and `value_code_sensitive` input attributes for
  setting sensitive USER_INPUT values. The plaintext is sent to meshStack as an embedded secret and is stored masked
  in Terraform state. Use these instead of `value_string`/`value_code` when the building block definition marks the
  input as sensitive. See the [known issue](https://feedback.meshcloud.io/knownissues/p/meshbuildingblock-api-update-requires-terraform-provider-upgrade-to-v0210)
  for migration guidance of existing configuration.

FEATURES:
- `meshstack_building_block_v2`: Added an optional `purge_on_delete` attribute (defaults to `false`). When set to `true`, deletion purges the Building Block from meshStack without running its configured deletion run, which is useful when a Building Block is stuck in a non-final state. Requires the `ADM_BUILDINGBLOCK_DELETE` permission.

## v0.20.13

FIXES:
- `meshstack_building_block_v2`: Reverted the change to `spec.inputs` and `status.outputs` from arrays to maps that was accidentally released in v0.20.12. This change requires meshStack 2026.23.0 or later and was not yet intended for release.

## v0.20.12

Requires meshStack 2026.23.0 or later (previously 2026.22.0).

BREAKING CHANGES:
- `meshstack_building_block_v2`: The `spec.inputs` and `status.outputs` fields have changed from arrays to maps in the upstream API.
  The internal client representation has been updated accordingly. No Terraform schema changes are required, but this requires meshStack 2026.23.0 or later.

FEATURES:
- New resource `meshstack_building_block_runner`: Manages a meshBuildingBlockRunner. Building block runners are agents that execute building block runs. Supports all implementation types (`TERRAFORM`, `GITHUB_WORKFLOW`, `GITLAB_PIPELINE`, `AZURE_DEVOPS_PIPELINE`, `MANUAL`), visibility restrictions, and optional workload identity federation configuration.

FIXES:
- `meshstack_building_block_v2`: `parent_building_blocks` is now treated as a set instead of a list to avoid ordering issues. The API may return `parent_building_blocks` in a different order than sent, which previously caused "Provider produced inconsistent result after apply" errors.

## v0.20.11

Requires meshStack 2026.22.0 or later (previously 2026.10.0).

BREAKING CHANGES:
- `meshstack_building_block_v2`: Renamed `target_ref.identifier` to `target_ref.name` to align with upstream API changes.
  Update configurations to use `name` instead of `identifier` when specifying the target workspace.
- `meshstack_workspace`: Renamed `ref.identifier` to `ref.name` to align with upstream API changes.
  Update any references to the workspace's `ref` field to use `name` instead of `identifier`.

## v0.20.10

FIXES:
- `meshstack_tag_definition`: Fixed a bug where `options` fields in `single_select` and `multi_select` value types could not be populated from Terraform expressions (e.g. `local.tag_options`). The internal representation has been changed from `[]types.String` to `types.List`, enabling correct handling of dynamic list references and proper propagation of diagnostics during configuration validation and resource creation.

## v0.20.9

BREAKING CHANGES:
- `meshstack_building_block_v2`: Building blocks in meshStack versions higher than v2026.20.0 are now soft-deleted instead of hard-deleted.
  The Terraform provider has been updated to correctly detect soft-deleted building blocks by inspecting the
  lifecycle state. To ensure correct deletion detection, upgrade the Terraform provider when using meshStack
  versions higher than v2026.20.0.

NOTE:
- This version is backwards-compatible and can be used with older versions of meshStack. If you are using a meshStack version
  v2026.20.0 or lower, the provider will continue to work as expected with hard-deleted building blocks.

## v0.20.8

BREAKING CHANGES:
- `meshstack_building_block_definition`:
  The field `supported_platforms` in the Client DTO is now aligned with an upstream API change.
  The resource schema itself is unaffected.
- `building_block_v2`: remove `created_on`, `marked_for_deletion_on`, `marked_for_deletion_by` from resource and datasource schema.

## v0.20.7

IMPROVEMENTS:
- Refactor HTTP client: extract auth into `Authorization` interface, add retry support with exponential backoff and `Retry-After` header parsing, and make request bodies replayable for retries.
- Add mutex to client secret authorization to prevent data races during concurrent token refresh.

## v0.20.6

BREAKING CHANGES:
- `meshstack_platform` (Azure): Remove `blueprint_service_principal` and `blueprint_location` fields from the replication config. These fields have been removed from the meshStack API.
- `meshstack_platform` (OpenShift): Remove `enable_template_instantiation` field from the replication config. This field has been removed from the meshStack API.
- `meshstack_landingzone`: Remove `openshift_template` field from OpenShift platform properties. This field has been removed from the meshStack API.

FEATURES:
- New `meshstack_api_key` resource for managing meshStack API keys with automatic secret rotation on expiry change.

## v0.20.5

FEATURES:
- `meshstack_building_block_definition`: Add `ref_name` field to the `azure_devops_pipeline` implementation block to specify the Git reference name (branch, tag, or commit) to use for the Azure DevOps pipeline run.

IMPROVEMENTS:
- Update dependencies, such as Go to 1.26

## v0.20.4

BREAKING CHANGES:
- Remove `api_version` and `kind` attributes from all resource and data source schemas:
  `meshstack_project`, `meshstack_tenant_v4`, `meshstack_workspace`, `meshstack_payment_method`,
  `meshstack_tag_definition`, `meshstack_building_block_v2`, `meshstack_buildingblock`,
  `meshstack_service_instance`, `meshstack_project_user_binding`, `meshstack_project_group_binding`,
  `meshstack_workspace_user_binding`, `meshstack_workspace_group_binding`.

FEATURES:
- New `meshstack_tenants` data source for listing tenants with optional workspace/project/platform filters.
- New `meshstack_building_block_definitions` data source for listing Building Block Definitions.
- Add `provider::meshstack::load_file`/`provider::meshstack::encode_file` functions for convenient Building Block definition `FILE` input.
  Improve related documentation.

IMPROVEMENTS:
- `meshstack_workspace`: Add computed `ref` field to resource and data source for use as cross-resource reference.
- `meshstack_tenant_v4`: Add computed `ref` field to resource for use as `target_ref` in `meshstack_building_block_v2`.
- `meshstack_platform`: Add computed `identifier` field (`<platform-name>.<location-name>`) to resource and data source
  suitable for direct use as `platform_identifier` in `meshstack_tenant_v4`.
- `meshstack_platform`: Expose `spec.access_information` on resource and data source, backed by platform
  `accessInformation` from the meshPlatform v2 API.
- Add broad provider test coverage with 26 new mock-based unit + acceptance test files across 15 resource/data source domains.
- Refactor test infrastructure into `internal/provider/acctest/testconfig/` with fluent immutable config operations and shared `xknownvalue` helpers.
- Add concise dependency-wiring guidance in examples and docs, including `one(...)` patterns and `version_latest`/`version_latest_release` usage.

## v0.20.3

BREAKING CHANGES:
- `meshstack_platform`: Remove the `allow_hierarchical_management_group_assignment` attribute from the `azurerg` platform
  config. This field was never functional in the meshStack API (always returned `false`) and has been removed from the
  meshStack API. Remove this attribute from your Terraform configurations if present.

## v0.20.2

FEATURES:
- `meshstack_building_block_definition`: Validate that the `symbol` field does not exceed 100 KiB when decoded from base64. Configurations exceeding this limit will receive a descriptive error during `plan`.

FIXES:
- `meshstack_building_block_definition`: Validate that `spec.description` does not exceed 255 characters.

## v0.20.1

FEATURES:
- Document `import` for all resources (also previously missing ones like `meshstack_building_block_definition`) via import block instead of shell script.<br>
  Note: Import is not yet supported for `meshstack_building_block` and `meshstack_building_block_v2`.

FIXES:
- `meshstack_building_block_definition`: Restrict the allowed `type` values for outputs to `STRING`, `CODE`, `INTEGER`, `BOOLEAN`. 
  The types `FILE`, `LIST`, `SINGLE_SELECT`, and `MULTI_SELECT` are only valid for inputs and were incorrectly accepted for outputs before.

## v0.20.0

Requires meshStack 2026.10.0 or later.

FEATURES:
- `meshstack_building_block_definition`: Add `pre_run_script` field to the `terraform` implementation block. 
- `meshstack_platform`: Add `aws_identity_store` optional block to the AWS replication config as an alternative to `aws_sso`. It uses the AWS Identity Store API directly and does not require a SCIM token, making it compatible with Workload Identity Federation (WIF). Setting both `aws_sso` and `aws_identity_store` is not allowed.

## v0.19.4

FIXES:
- Properly support specifying non-null `secret_version` along `secret_value`.
- Document `nonsensitive(sha256(...))` workaround for non-ephemeral `secret_value` inputs.

## v0.19.3

FEATURES:
- Add `provider::meshstack::load_image_file` function for convenient Building Block symbol loading

FIXES:
- `meshstack_building_block_definition`: Require replace if `spec.target_type` is changed.
- `meshstack_building_block_definition`: `use_mesh_http_backend_fallback` for Terraform implementation defaults to `true`.
- `meshstack_building_block_definition`: Validate `notification_subscribers` to start with `user:` or `email:`.

## v0.19.2

FIXES:
- `meshstack_building_block_definition`: Provide empty defaults for `version_spec.outputs`, `version_spec.dependency_refs`, `metadata.tags`, `spec.notification_subscribers` to match backend API behavior better.
- Plan modification of secrets which are added after resource creation as a resource update 

## v0.19.1

BREAKING CHANGES:
- `meshstack_platform`: `tag_mappers` changed their schema type from list to set.

## v0.19.0

FEATURES:
- The following resources are now generally available (GA) and fully supported for production use:
`meshstack_platform`, `meshstack_landingzone`, `meshstack_platform_type`, `meshstack_location`, and `meshstack_integrations`.
These resources have been thoroughly tested and validated, and are now considered stable and ready for production deployments.
  - `meshstack_platform`: Platform API now uses GA version `v2` (was `v2-preview`).
  - `meshstack_landingzone`: Landing zone API now uses GA version `v1` (was `v1-preview`).
  - `meshstack_platform_type`: Platform type API now uses GA version `v1` (was `v1-preview`).
  - `meshstack_location`: Location API now uses GA version `v1` (was `v1-preview`).
  - `meshstack_integrations`: Integration API now uses GA version `v1` (was `v1-preview`).

FIXES:
- `meshstack_platform`: `aws_sso.sign_in_url` is now a required attribute in the AWS SSO configuration.
  Existing configurations must be updated to explicitly provide a valid `sign_in_url` value.
- `datasource.meshstack_integrations`: Properly mark secrets as read-only.

BREAKING CHANGES:
- `meshstack_location`: Added required `owned_by_workspace` field to metadata. This field specifies the workspace that owns the location and must be provided when creating or updating locations.
- Removed `api_version` and `kind` fields from resource and data source schemas for `meshstack_platform`, `meshstack_platform_type`, and `meshstack_location`. These internal fields are no longer exposed to Terraform users. Existing configurations that reference these attributes must be updated to remove those references.
- `meshstack_platform`: Secrets are now write-only `secret_value` attributes instead of `plaintext`.
  Change the attribute from `plaintext` to `secret_value` in existing configs, and consider using ephemeral resources.
  Likewise, there's an additional `secret_version` attribute for secret rotation, and the read-only hash attribute has changed to `secret_hash`.
- `meshstack_platform`: The following attributes changed their schema type from list to set: `contributing_workspaces`, `restricted_to_workspaces`, `quota_definitions`, `role_mappings`, and `tenant_tags`.
  Configurations that rely on element ordering or index-based access (e.g., `quota_definitions[0]`) must be updated, as sets are unordered and do not support stable indexing.


## v0.18.2

FEATURES:

- `meshstack_service_instance` and `meshstack_service_instances`: Added support for `parameters` field in service instance specification.

FIXES:

- `meshstack_building_block_definition`: Handle `notification_subscribers` being filtered by backend
- `meshstack_building_block_definition`: Ignore empty vs. nil/null map/slices mismatch backend vs. config

## v0.18.1

FIXES:

- `meshstack_platform`: Adds missing `update_subscription_name` field to Azure config.
- `meshstack_platform`: `blueprint_location` is required for Azure.

## v0.18.0

FEATURES:

- New resource: `meshstack_building_block_definition`
- New resource: `meshstack_integration`

FIXES:

- `meshstack_platform`: Support custom platform in datasource as well

BREAKING CHANGES:

- Proper secret handling for data source `meshstack_integrations`,
  as only the hash is returned from the API now.

## v0.17.5

FEATURES:

- `meshstack_landingzone`: Added support for `custom` platform type.

## v0.17.4

FEATURES:

- `meshstack_platform`: Added support for `custom` platforms.
- Data source for `meshstack_service_instance` and `meshstack_service_instances`.

## v0.17.3

BREAKING CHANGES:

- Platform type resource: Added required `owned_by_workspace` field to metadata. This field specifies the workspace that owns the platform type and must be provided when creating or updating platform types.

## v0.17.2

FIXES:
- `meshstack_platform`: correctly mark `subscription_creation_error_cooldown_sec` as computed for Azure platforms.

## v0.17.1

FIXES:
- `meshstack_platform`: correctly handle null values of `subscription_creation_error_cooldown_sec` for Azure platforms.
- `meshstack_platform`: for Azure platforms using `customer_agreement`, `source_service_principal` is a required field.

OTHER:
- Improve documentation around Azure workload identity federation.

## v0.17.0

FEATURES:

- New data source: `meshstack_platform_types` for listing all platform types available in the meshStack installation.
- New data source: `meshstack_platform_type` for reading a single platform type by name.
- New resource: `meshstack_platform_type` for managing meshStack platform types.
- Added support for authenticating directly via `apitoken` or `MESHSTACK_API_TOKEN`, bypassing the initial login call.
- Checks if provider version is compatible with meshStack version.

FIXES:

- Platform resource: Added schema-level validation to ensure exactly one of `credential` or `workload_identity` is provided when `workload_identity` can be set.
- DELETE endpoints now properly include versioning Accept header.
- Context is now correctly propagated through client HTTP transport.

OTHER:
- Large internal refactoring disentangling client package from provider implementation.


## v0.16.6

FIXES:

- Landing zone resource and data source: Added support for `mandatory_building_block_refs` and `recommended_building_block_refs` fields.

## v0.16.5

BREAKING CHANGES:

- Landing zone: Added required `owned_by_workspace` field to metadata. This field specifies the workspace that owns the landing zone and must be provided when creating or updating landing zones.

## v0.16.4

FEATURES:

- New resource: `meshstack_location` for managing meshStack locations.
- Start implementing acceptance tests running against local meshStack

FIXES:

- Tenant v4 resource: More granular plan modifiers to reduce unnecessary recreations.
- Tenant v4 resource: Set `wait_for_completion` during import.

OTHER:
- Updated to Go 1.25 including nix environment
- Fix lint issues and enforce them in CI

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
