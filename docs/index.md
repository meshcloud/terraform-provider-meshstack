---
page_title: "meshStack Provider"
description: |-
  Manage meshStack resources.
  
---

# meshStack Provider

The meshStack terraform provider is an open-source tool, licensed under the MPL-2.0, and is actively maintained by meshcloud GmbH. The provider leverages external APIs of meshStack to manage resources as code.


## Example Usage

```terraform
provider "meshstack" {
  endpoint  = "meshfed.url"
  apikey    = "API_KEY"
  apisecret = "API_SECRET"
}
```

### Required

- `apikey` (String) API Key to authenticate against the meshStack API
- `apisecret` (String) API Secret to authenticate against the meshStack API
- `endpoint` (String) URl of meshStack API, e.g. `https://api.my.meshstack.io`
