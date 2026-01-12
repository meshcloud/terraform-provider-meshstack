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

Note: `task` is also available via `nix`, for example
```bash
nix develop --command task testacc
```

Debugging can be enabled by setting `TF_LOG=DEBUG` or `TF_ACC_LOG=DEBUG` (when running tests), 
which shows all full HTTP request and response communication.

## Running Tests

This project uses [Task](https://taskfile.dev) for common development workflows. 
The available tasks can be found in `Taskfile.yml`.

### Acceptance Tests

Acceptance tests run against a real meshStack API and require environment variables to be configured in a `.env` file:

```bash
# Run all acceptance tests
task testacc

# Run specific acceptance test(s) by name pattern
task testacc -- -run=BuildingBlockDefinition

# Run multiple specific tests
task testacc -- -run=BuildingBlock|Workspace
```

### Unit Tests

```bash
# Run unit tests only (excludes acceptance tests)
task test

# Run specific unit test(s)
task test -- -run=TestValidation
```

### Other Development Tasks

```bash
# Build the provider
task build

# Install provider locally
task install

# Run linter (also checks formatting)
task lint

# Fix formatting and linting issues
task lint -- --fix

# Generate documentation
task generate

# Clean build artifacts
task clean
```

## Code Formatting

This project uses golangci-lint with the gci formatter to enforce consistent import ordering:

1. Go standard library imports
2. External dependencies (third-party packages)
3. Local modules (this repository's packages)

Each section is separated by a blank line. To format your code, run:

```bash
task lint -- --fix
```
