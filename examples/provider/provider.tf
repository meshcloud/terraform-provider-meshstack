# A browser login, stored in a meshStack CLI profile by `meshstack login`. The profile carries
# the endpoint and the credential. The workspace picks the role this configuration acts with.
provider "meshstack" {
  profile   = "my-profile"
  workspace = "my-workspace-ab12c"
}
