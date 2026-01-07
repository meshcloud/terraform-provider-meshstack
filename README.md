# meshStack Terraform Provider

This is the repository for the meshStack Terraform Provider, which allows one to use Terraform with meshStack by meshcloud. Learn more about meshcloud at https://www.meshcloud.io. This provider is officially registered and documented under [terraform registry](https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs).

For general information about Terraform, visit the [official website](https://www.terraform.io).

## Support, Bugs, Feature Requests

Please submit support questions via email to support@meshcloud.io. Support questions submitted under the Issues section of this repo will be handled on a "best effort" basis.

Feature requests can be submitted at [canny.io](https://meshcloud.canny.io).

## Local Development

To use the provider locally during development place the following in `~/.terraformrc`:

```
provider_installation {

  dev_overrides {
      "meshcloud/meshstack" = "<GOBIN>",
      "registry.terraform.io/meshcloud/meshstack" = "<GOBIN>"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Replace `<GOBIN>` with the output of `go env GOBIN` or `go env GOPATH` + `/bin`.
Run `go install` to update your local provider installation.
If everything is working correctly Terraform will show a warning that dev overrides are being used.

## Running Tests

This project uses [Task](https://taskfile.dev) for common development workflows. The available tasks can be found in `Taskfile.yml`.

### Acceptance Tests

Acceptance tests run against a real meshStack API and require environment variables to be configured in a `.env` file:

```bash
# Run all acceptance tests
go-task testacc

# Run specific acceptance test(s) by name pattern
go-task testacc TESTARGS="-run=BuildingBlockDefinition"

# Run multiple specific tests
go-task testacc TESTARGS="-run=BuildingBlock|Workspace"
```

### Unit Tests

```bash
# Run unit tests only (excludes acceptance tests)
go-task test
```

### Other Development Tasks

```bash
# Build the provider
go-task build

# Install provider locally
go-task install

# Format code
go-task fmt

# Run linter
go-task lint

# Generate documentation
go-task generate

# Clean build artifacts
go-task clean
```
