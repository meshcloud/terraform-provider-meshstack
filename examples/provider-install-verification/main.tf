terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  endpoint  = "https://federation.dev.meshcloud.io"
  apikey    = "c0da6389-217a-4581-bed0-2728a1fed78b"
  apisecret = "tnPKGcVoFKeaCXRyknRfx0V2xXL7bKbo"
}

data "meshstack_buildingblock" "test" {
  metadata = {
    uuid = "a797d382-b316-4827-95a1-5af8e6a1217f"
  }
}

output "bb_provider_uuid" {
  value = data.meshstack_buildingblock.test.metadata.definition_uuid
}
