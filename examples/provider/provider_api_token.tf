# An API token, which is what a building block run gets injected as MESHSTACK_API_TOKEN. Export it
# rather than writing it here, so that it reaches neither the configuration nor terraform state.
provider "meshstack" {
  endpoint = "https://api.my.meshstack.io"
  apitoken = "API_TOKEN"
}
