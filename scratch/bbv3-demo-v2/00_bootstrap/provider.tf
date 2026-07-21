# Bootstrap provider — the DEFAULT provider here is admin, reading endpoint + credentials from the
# environment (MESHSTACK_ENDPOINT / MESHSTACK_API_KEY / MESHSTACK_API_SECRET, sourced from .env).
#
# meshstack comes from the dev_overrides CLI config (see README), so no init is needed for it; the
# small hashicorp/random provider IS installed normally, so run `tofu init` once in this folder
# before the first apply. The operator/ and appteam/ states use only meshstack and need no init.

terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

provider "meshstack" {
  # endpoint / apikey / apisecret inherited from MESHSTACK_* environment variables (admin).
}
