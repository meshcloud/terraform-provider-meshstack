# Find a group that your identity provider created through SCIM by its ID there,
# such as the object ID of an Entra ID group.
data "meshstack_workspace_user_groups" "developers" {
  egid = "5f0c2a8e-7d1b-4c3a-9e6f-2b8d4a1c7e93"
}
