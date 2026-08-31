---
page_title: "meshStack Provider"
description: |-
  Manage meshStack resources.
  
---

# meshStack Provider

The meshStack terraform provider is an open-source tool, licensed under the MPL-2.0, and is actively maintained by meshcloud GmbH. The provider exposes APIs of meshStack to manage resources as code.

**Note:** This provider version requires meshStack version 2026.35.0 or higher. The provider automatically validates version compatibility during initialization.

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
     └─ meshstack_building_block.spec.building_block_definition_version_ref

Tenant BB dependency chain
meshstack_workspace
  └─ meshstack_location
     └─ meshstack_platform (identifier)
        └─ meshstack_landingzone
           └─ meshstack_tenant
              └─ meshstack_building_block (target_ref)
```

## Authentication

The provider reads its configuration from three places, highest first: the `provider` block, the
`MESHSTACK_*` environment variables, and a **meshStack CLI profile**. A profile is a named bundle of
endpoint, credential and default workspace, written by `meshstack auth login`, so a block naming
only a profile is a complete configuration. The meshStack CLI documentation is the single
description of the resolution order.

A profile ranks below the environment and is never an override, because a `terraform plan` whose
result depends on what is in the operator's home directory is not reproducible. Where a profile is
picked by matching its endpoint rather than being named, the provider says so in a log record, which
`TF_LOG=WARN` shows.

The provider never opens a browser. It refreshes a profile's existing browser login and writes the
rotated refresh token back — it has to, because losing it ends the session the meshStack CLI shares
— but creating one is `meshstack login`.

## Example Usage

```terraform
# A meshStack CLI profile: the endpoint, the credential and the default workspace all come from it.
provider "meshstack" {
  profile = "meshstack-dev"
}

# An API key. Prefer the MESHSTACK_API_KEY and MESHSTACK_API_SECRET environment variables, or a
# profile, over writing a secret into a configuration file.
provider "meshstack" {
  endpoint  = "meshfed.url"
  apikey    = "API_KEY"
  apisecret = "API_SECRET"
}

# An API token, which is what a building block run gets injected as MESHSTACK_API_TOKEN.
provider "meshstack" {
  endpoint = "meshfed.url"
  apitoken = "API_TOKEN"
}
```

## Schema

### Optional

- `endpoint` (String) URL of the meshStack API, e.g. `https://api.my.meshstack.io`. Required unless a profile supplies it.
- `profile` (String) meshStack CLI profile to authenticate with.
- `workspace` (String) Workspace this provider acts in. Required for a profile holding a browser login, because a meshStack user access token is bound to one workspace; an API key carries its own.
- `apikey` (String) Id of the API key to authenticate with. Required if neither `apitoken` nor a profile is set.
- `apisecret` (String, Sensitive) Secret of that API key. Required if neither `apitoken` nor a profile is set.
- `apitoken` (String, Sensitive) An access token obtained elsewhere. Nothing can refresh one, so it expires during long-running work. Required if neither `apikey` and `apisecret` nor a profile is set.
