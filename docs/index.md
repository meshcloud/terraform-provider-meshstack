---
page_title: "meshStack Provider"
description: |-
  Manage meshStack resources.
  
---

# meshstack Provider

This provider is still in its' early stages and under heavy development. It's not considered ready for production use.


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
