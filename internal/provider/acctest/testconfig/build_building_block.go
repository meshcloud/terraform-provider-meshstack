package testconfig

import (
	"testing"
)

// BBWorkspace builds a workspace, a building block definition owned by it, and a v3 building
// block on that workspace wired to the definition's latest version. workspaceAddr is the
// underlying Workspace(t) address, returned so callers can attach further workspace-scoped
// resources without rebuilding the workspace.
func BBWorkspace(t *testing.T) (config Config, buildingBlockAddr Traversal, buildingBlockDefinitionAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	paymentMethodConfig, setPaymentMethodInput := PaymentMethodInput(t, workspaceAddr)
	buildingBlockDefinitionConfig := Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
	)
	return Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		// Wire only the version uuid (not the whole version_latest object, which carries
		// content_hash). content_hash is a TF-only field that can't be recovered on import,
		// so leaving it unset keeps ImportBlockWithID a no-op. Tests that exercise content_hash
		// set it explicitly in later steps.
		Descend("spec", "building_block_definition_version_ref")(SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest", "uuid"))),
		Descend("spec", "target_ref")(SetAddr(workspaceAddr, "ref")),
		setPaymentMethodInput,
	).Join(workspaceConfig, paymentMethodConfig, buildingBlockDefinitionConfig), buildingBlockAddr, buildingBlockDefinitionAddr, workspaceAddr
}

// PaymentMethodInput builds a payment method in the workspace, and the consumer that sets it as the
// payment_method input of a building block from resource_01_workspace.tf.
func PaymentMethodInput(t *testing.T, workspaceAddr Traversal) (paymentMethodConfig Config, setPaymentMethodInput ExpressionConsumer) {
	t.Helper()
	paymentMethodConfig, paymentMethodAddr := PaymentMethod(t, workspaceAddr)
	return paymentMethodConfig, Descend("spec", "inputs", "payment_method", "value")(SetRawExpr(`jsonencode(%s)`, paymentMethodAddr.Join("ref")))
}

// BBWorkspaceParentChild builds a workspace, two building block definitions owned by it, and a v3
// building block from each, the child referencing the parent through spec.parent_building_block_refs.
// The child's definition declares the parent's definition in version_spec.dependency_refs, which
// meshStack requires of any parent.
func BBWorkspaceParentChild(t *testing.T) (config Config, parentAddr Traversal, childAddr Traversal) {
	t.Helper()
	workspaceConfig, definition, buildingBlock := workspaceParentChildBuilders(t)
	parentDefinitionConfig, parentDefinitionAddr := definition("parent")
	childDefinitionConfig, childDefinitionAddr := definition("child",
		Descend("version_spec", "dependency_refs")(SetRawExpr("[%s]", parentDefinitionAddr.Join("ref"))),
	)

	parentConfig, parentAddr := buildingBlock("parent", parentDefinitionAddr)
	childConfig, childAddr := buildingBlock("child", childDefinitionAddr,
		Descend("spec", "parent_building_block_refs")(SetRawExpr("[%s]", parentAddr.Join("ref"))),
	)
	return childConfig.Join(parentConfig, workspaceConfig, parentDefinitionConfig, childDefinitionConfig), parentAddr, childAddr
}

// BBWorkspaceDraftParents builds two draft building block definitions, two parent building blocks
// from the first one and a child from the second one under parent A. Moving the child to parent B
// keeps the definition's dependency filled.
func BBWorkspaceDraftParents(t *testing.T) (config Config, parentAAddr Traversal, parentBAddr Traversal, childAddr Traversal) {
	t.Helper()
	workspaceConfig, definition, buildingBlock := workspaceParentChildBuilders(t)
	draft := Descend("version_spec", "draft")(SetRawExpr("true"))
	parentDefinitionConfig, parentDefinitionAddr := definition("parent", draft)
	childDefinitionConfig, childDefinitionAddr := definition("child", draft,
		Descend("version_spec", "dependency_refs")(SetRawExpr("[%s]", parentDefinitionAddr.Join("ref"))),
	)

	parentAConfig, parentAAddr := buildingBlock("parent_a", parentDefinitionAddr)
	parentBConfig, parentBAddr := buildingBlock("parent_b", parentDefinitionAddr)
	childConfig, childAddr := buildingBlock("child", childDefinitionAddr,
		Descend("spec", "parent_building_block_refs")(SetRawExpr("[%s]", parentAAddr.Join("ref"))),
	)
	return childConfig.Join(parentAConfig, parentBConfig, workspaceConfig, parentDefinitionConfig, childDefinitionConfig),
		parentAAddr, parentBAddr, childAddr
}

type (
	definitionBuilder    func(name string, extra ...ExpressionConsumer) (definitionConfig Config, definitionAddr Traversal)
	buildingBlockBuilder func(name string, definitionAddr Traversal, extra ...ExpressionConsumer) (buildingBlockConfig Config, buildingBlockAddr Traversal)
)

// workspaceParentChildBuilders builds definitions and building blocks from the _01_workspace example in
// one workspace. Each call names its own resource label, so one example file yields several of each.
func workspaceParentChildBuilders(t *testing.T) (workspaceConfig Config, definition definitionBuilder, buildingBlock buildingBlockBuilder) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	paymentMethodConfig, setPaymentMethodInput := PaymentMethodInput(t, workspaceAddr)

	definition = func(name string, extra ...ExpressionConsumer) (definitionConfig Config, definitionAddr Traversal) {
		return Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
			append([]ExpressionConsumer{
				RenameKey(name),
				ExtractAddress(&definitionAddr),
				OwnedByWorkspace(workspaceAddr),
				Descend("spec", "display_name")(SetString("Test BB v3 " + name + " Definition")),
			}, extra...)...,
		), definitionAddr
	}

	buildingBlock = func(name string, definitionAddr Traversal, extra ...ExpressionConsumer) (buildingBlockConfig Config, buildingBlockAddr Traversal) {
		return Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			append([]ExpressionConsumer{
				RenameKey(name),
				ExtractAddress(&buildingBlockAddr),
				Descend("spec", "display_name")(SetString("my-" + name + "-building-block")),
				// Reference only the version uuid, for the reason given in BBWorkspace.
				Descend("spec", "building_block_definition_version_ref")(SetRawExpr(`{ uuid = %s }`, definitionAddr.Join("version_latest", "uuid"))),
				Descend("spec", "target_ref")(SetAddr(workspaceAddr, "ref")),
				setPaymentMethodInput,
			}, extra...)...,
		), buildingBlockAddr
	}
	return workspaceConfig.Join(paymentMethodConfig), definition, buildingBlock
}

// BBTenant builds a workspace (+project/platform/landing-zone/tenant) and a v3 building block
// targeting that tenant. The building block definition uses the terraform implementation and
// clones terraformRepoUrl. workspaceAddr is returned so callers can attach further workspace-scoped
// resources without rebuilding the workspace.
func BBTenant(t *testing.T, terraformRepoUrl string) (config Config, buildingBlockAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	var tenantAddr Traversal
	tenantConfig := Resource{Name: "tenant"}.Config(t).FirstBlockOnly().WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("spec")(
			Descend("platform_ref")(SetRawExpr("%s", platformAddr.Join("ref"))),
			Descend("landing_zone_ref")(SetAddr(landingZoneAddr, "ref")),
		),
	)

	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "supported_platforms")(SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
		Descend("version_spec", "implementation", "terraform", "repository_url")(SetRawExpr("%q", terraformRepoUrl)),
	)

	return Resource{Name: "building_block", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		// Wire only the version uuid (see BBWorkspace) so content_hash stays unset and
		// ImportBlockWithID remains a no-op.
		Descend("spec", "building_block_definition_version_ref")(SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest", "uuid"))),
		Descend("spec", "target_ref")(SetAddr(tenantAddr, "ref")),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr, workspaceAddr
}
