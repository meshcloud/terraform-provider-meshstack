# meshStack Terraform Provider

The official [Terraform](https://www.terraform.io) provider for [meshStack](https://www.meshcloud.io)
by meshcloud. It lets you manage meshStack resources — workspaces, projects, building blocks, tenants,
and more — as code.

The provider is published on the Terraform Registry, with full documentation and examples:
**[registry.terraform.io/providers/meshcloud/meshstack](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs)**.

## Usage

Declare the provider and let Terraform pull the released version from the registry:

```hcl
terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  endpoint  = "https://your.meshstack.example" # or MESHSTACK_ENDPOINT
  apikey    = "..."                            # or MESHSTACK_API_KEY
  apisecret = "..."                            # or MESHSTACK_API_SECRET
}
```

See the [registry documentation](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs)
for the full list of resources, data sources, and example configurations.

## Support, bugs, feature requests

- **Support questions**: email support@meshcloud.io. Questions filed as GitHub issues are handled on a
  best-effort basis.
- **Feature requests**: [feedback.meshcloud.io](https://feedback.meshcloud.io).

## Contributing / development

Building the provider from source, running tests, and adding resources are covered in
[`DEVELOPMENT.md`](DEVELOPMENT.md) (and the always-on conventions in [`AGENTS.md`](AGENTS.md)).
