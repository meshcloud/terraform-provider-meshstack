# App-team state — the DEFAULT provider here authenticates as the APP TEAM, using the least-privilege
# scoped key from 00_bootstrap (fed in via TF_VAR_appteam_client_id / _secret). Endpoint inherited
# from MESHSTACK_ENDPOINT. Only meshstack is used (dev_overrides), so no `tofu init`.

terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  apikey    = var.appteam_client_id
  apisecret = var.appteam_client_secret
  # endpoint inherited from MESHSTACK_ENDPOINT.
}
