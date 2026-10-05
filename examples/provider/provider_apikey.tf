# An API key, with its secret exported as MESHSTACK_API_SECRET rather than written here, so that it
# reaches neither the configuration nor terraform state.
provider "meshstack" {
  endpoint = "https://api.my.meshstack.io"
  apikey   = "API_KEY"
}
