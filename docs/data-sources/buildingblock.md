---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "meshstack_buildingblock Data Source - terraform-provider-meshstack"
subcategory: ""
description: |-
  Query a single Building Block by UUID.
---

# meshstack_buildingblock (Data Source)

Query a single Building Block by UUID.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `metadata` (Attributes) Building Block metadata. UUID of the target Building Block must be set here. (see [below for nested schema](#nestedatt--metadata))

### Read-Only

- `api_version` (String) Building Block datatype version
- `kind` (String) meshObject type, always `meshBuildingBlock`.
- `spec` (Attributes) Building Block specification. (see [below for nested schema](#nestedatt--spec))
- `status` (Attributes) Current Building Block status. (see [below for nested schema](#nestedatt--status))

<a id="nestedatt--metadata"></a>
### Nested Schema for `metadata`

Required:

- `uuid` (String)

Read-Only:

- `created_on` (String)
- `definition_uuid` (String)
- `definition_version` (Number)
- `force_purge` (Boolean)
- `marked_for_deletion_by` (String)
- `marked_for_deletion_on` (String)
- `tenant_identifier` (String)


<a id="nestedatt--spec"></a>
### Nested Schema for `spec`

Read-Only:

- `display_name` (String)
- `inputs` (Attributes List) List of Building Block inputs. (see [below for nested schema](#nestedatt--spec--inputs))
- `parent_building_blocks` (Attributes List) (see [below for nested schema](#nestedatt--spec--parent_building_blocks))

<a id="nestedatt--spec--inputs"></a>
### Nested Schema for `spec.inputs`

Read-Only:

- `key` (String)
- `value` (String)
- `value_type` (String)


<a id="nestedatt--spec--parent_building_blocks"></a>
### Nested Schema for `spec.parent_building_blocks`

Read-Only:

- `buildingblock_uuid` (String)
- `definition_uuid` (String)



<a id="nestedatt--status"></a>
### Nested Schema for `status`

Read-Only:

- `outputs` (Attributes List) List of building block outputs. (see [below for nested schema](#nestedatt--status--outputs))
- `status` (String) Execution status. One of `WAITING_FOR_DEPENDENT_INPUT`, `WAITING_FOR_OPERATOR_INPUT`, `PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`.

<a id="nestedatt--status--outputs"></a>
### Nested Schema for `status.outputs`

Read-Only:

- `key` (String)
- `value` (String)
- `value_type` (String)