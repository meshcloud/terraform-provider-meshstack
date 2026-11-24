#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# tfplugindocs reads the schema under registry.terraform.io/hashicorp/<name> only.
address=registry.terraform.io/hashicorp/meshstack

go build -o "$work/terraform-provider-meshstack" .
cat >"$work/main.tf" <<HCL
terraform {
  required_providers {
    meshstack = { source = "$address" }
  }
}
HCL
cat >"$work/tofurc" <<HCL
provider_installation {
  dev_overrides {
    "$address" = "$work"
  }
}
HCL

TF_CLI_CONFIG_FILE="$work/tofurc" tofu -chdir="$work" providers schema -json >"$work/schema.json"
go tool tfplugindocs generate --providers-schema "$work/schema.json"
