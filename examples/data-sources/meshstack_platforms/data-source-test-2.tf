# The listing the consumer workspace runs, through the meshstack-other provider alias backed by its
# restricted API key. Filtered to the operator's workspace, so both P_pub and P_priv would match on
# ownership and only entitlement decides what comes back.

data "meshstack_platforms" "published" {
  provider = meshstack-other

  owned_by_workspace = meshstack_workspace.example.metadata.name
  publication_state  = "PUBLISHED"
}
