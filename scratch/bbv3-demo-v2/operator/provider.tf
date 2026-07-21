# Operator state — the DEFAULT provider here authenticates as the PLATFORM OPERATOR, using the
# scoped key minted in 00_bootstrap (fed in via TF_VAR_operator_client_id / _secret). The endpoint is
# still inherited from MESHSTACK_ENDPOINT. Only meshstack is used (dev_overrides), so no `tofu init`.

terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  apikey    = var.operator_client_id
  apisecret = var.operator_client_secret
  # endpoint inherited from MESHSTACK_ENDPOINT.
}
