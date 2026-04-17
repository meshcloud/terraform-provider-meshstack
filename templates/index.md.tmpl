---
page_title: "meshStack Provider"
description: |-
  Manage meshStack resources.
  
---

# meshStack Provider

The meshStack terraform provider is an open-source tool, licensed under the MPL-2.0, and is actively maintained by meshcloud GmbH. The provider exposes APIs of meshStack to manage resources as code.

**Note:** This provider version requires meshStack version __MIN_MESHSTACK_VERSION__ or higher. The provider automatically validates version compatibility during initialization.

## Dependency wiring patterns

Many meshStack resources depend on other meshStack objects. Prefer reusable references from data sources and computed outputs:

- Prefer plural data sources with `one(...)` when your filter selects exactly one match.
- Use singular data sources mainly for existence checks where `metadata.uuid` is enough.
- Reuse computed outputs such as `ref`, `identifier`, `version_latest`, and `version_latest_release` to avoid hardcoded identifiers.

Example dependency graphs:

```text
BBD -> BB
meshstack_building_block_definition
  └─ version_latest / version_latest_release
     └─ meshstack_building_block_v2.spec.building_block_definition_version_ref

Tenant BB dependency chain
meshstack_workspace
  └─ meshstack_location
     └─ meshstack_platform (identifier)
        └─ meshstack_landingzone
           └─ meshstack_tenant_v4
              └─ meshstack_building_block_v2 (target_ref)
```

## Example Usage

```terraform
# Using API Key and Secret
provider "meshstack" {
  endpoint  = "meshfed.url"
  apikey    = "API_KEY"
  apisecret = "API_SECRET"
}

# Using API Token
provider "meshstack" {
  endpoint = "meshfed.url"
  apitoken = "API_TOKEN"
}
```

## Schema

### Required

- `endpoint` (String) URL of meshStack API, e.g. `https://api.my.meshstack.io`. Can be sourced from `MESHSTACK_ENDPOINT`.

### Optional

- `apikey` (String) API Key to authenticate against the meshStack API. Can be sourced from `MESHSTACK_API_KEY`. Required if `apitoken` is not set.
- `apisecret` (String) API Secret to authenticate against the meshStack API. Can be sourced from `MESHSTACK_API_SECRET`. Required if `apitoken` is not set.
- `apitoken` (String) API Token to authenticate against the meshStack API. Can be sourced from `MESHSTACK_API_TOKEN`. Required if `apikey` and `apisecret` are not set.
