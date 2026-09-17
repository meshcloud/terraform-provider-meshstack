package provider

// subjectTemplateDescription documents the placeholder syntax of a runner's subject template. The
// placeholder set is a public contract for self-hosted runner operators, so the description spells
// it out in full; lead is the sentence that names the attribute.
func subjectTemplateDescription(lead string) string {
	return lead + ", for example `system:serviceaccount:my-namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}`. " +
		"Two placeholders are available: `{{ workspaceIdentifier }}` (the identifier of the workspace owning the building block definition) and " +
		"`{{ buildingBlockDefinitionUuid }}` (`metadata.uuid` of that definition). A template without placeholders is a fixed subject, " +
		"which is what a runner minting one identity for all its runs declares. " +
		"meshStack resolves the template per building block definition and reports the result as " +
		"`meshstack_building_block_definition.<name>.status.workload_identity_federation.subject`."
}
