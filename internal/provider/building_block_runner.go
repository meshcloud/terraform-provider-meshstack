package provider

const SharedBuildingBlockRunnerUuid = "98520496-627d-43e6-82da-ce499179ff3f"

// subjectTemplateDescription documents the placeholder syntax of a runner's subject template. The
// placeholder set is a public contract for self-hosted runner operators, so the description spells
// it out in full; lead is the sentence that names the attribute.
func subjectTemplateDescription(lead string) string {
	return lead + ", for example `system:serviceaccount:my-namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}`. " +
		"Placeholders: `{{ workspaceIdentifier }}` (workspace owning the building block definition) and `{{ buildingBlockDefinitionUuid }}` (its `metadata.uuid`). " +
		"Without placeholders it is a fixed subject. meshStack fills it in per definition and reports the result as " +
		"`meshstack_building_block_definition.<name>.version_latest.workload_identity_federation.subject`."
}
