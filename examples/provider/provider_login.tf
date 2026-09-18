# A browser login, stored in a meshStack CLI profile by `meshstack auth login`. The profile carries
# the endpoint and the credential, and the workspace says which one this configuration acts in.
provider "meshstack" {
  profile   = "my-profile"
  workspace = "my-workspace-ab12c"
}
