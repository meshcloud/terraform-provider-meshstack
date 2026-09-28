# The throwaway RSA public key the backend parses spec.public_key with. It lives in the test's
# testdata rather than here because the documented example carries a truncated placeholder, which the
# backend would reject.

variable "runner_public_key" {
  type = string
}

# The subject template of the workload identity federation example; a case dials it to see the runner
# update in place.
variable "subject_template" {
  type    = string
  default = "system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"
}
