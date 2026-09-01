# A meshStack CLI profile: the endpoint, the credential and the default workspace all come from
# it, so nothing sensitive is in the configuration. `meshstack auth login --profile dev` writes one.
provider "meshstack" {
  profile = "dev"
}

# An API key. Prefer the MESHSTACK_API_KEY and MESHSTACK_API_SECRET environment variables, or a
# profile, over writing a secret into a configuration file.
provider "meshstack" {
  endpoint  = "meshfed.url"
  apikey    = "API_KEY"
  apisecret = "API_SECRET"
}
