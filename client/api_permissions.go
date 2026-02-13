package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

// API Permissions as defined in https://docs.meshcloud.io/api/authentication/api-permissions/

type ApiPermission string

// Workspace Permissions (non-admin).
var (
	WorkspacePermissions = enum.Enum[ApiPermission]{}

	PermissionBuildingBlockDefinitionDelete = WorkspacePermissions.Entry("BUILDINGBLOCKDEFINITION_DELETE")
	PermissionBuildingBlockDefinitionList   = WorkspacePermissions.Entry("BUILDINGBLOCKDEFINITION_LIST")
	PermissionBuildingBlockDefinitionSave   = WorkspacePermissions.Entry("BUILDINGBLOCKDEFINITION_SAVE")

	PermissionBuildingBlockRunnerDelete = WorkspacePermissions.Entry("BUILDINGBLOCKRUNNER_DELETE")
	PermissionBuildingBlockRunnerList   = WorkspacePermissions.Entry("BUILDINGBLOCKRUNNER_LIST")
	PermissionBuildingBlockRunnerSave   = WorkspacePermissions.Entry("BUILDINGBLOCKRUNNER_SAVE")

	PermissionBuildingBlockDelete = WorkspacePermissions.Entry("BUILDINGBLOCK_DELETE")
	PermissionBuildingBlockList   = WorkspacePermissions.Entry("BUILDINGBLOCK_LIST")
	PermissionBuildingBlockSave   = WorkspacePermissions.Entry("BUILDINGBLOCK_SAVE")

	PermissionCommunicationDefinitionDelete = WorkspacePermissions.Entry("COMMUNICATIONDEFINITION_DELETE")
	PermissionCommunicationDefinitionList   = WorkspacePermissions.Entry("COMMUNICATIONDEFINITION_LIST")
	PermissionCommunicationDefinitionSave   = WorkspacePermissions.Entry("COMMUNICATIONDEFINITION_SAVE")

	PermissionCommunicationDelete = WorkspacePermissions.Entry("COMMUNICATION_DELETE")
	PermissionCommunicationList   = WorkspacePermissions.Entry("COMMUNICATION_LIST")
	PermissionCommunicationSave   = WorkspacePermissions.Entry("COMMUNICATION_SAVE")

	PermissionEventLogList = WorkspacePermissions.Entry("EVENTLOG_LIST")

	PermissionIntegrationDelete = WorkspacePermissions.Entry("INTEGRATION_DELETE")
	PermissionIntegrationList   = WorkspacePermissions.Entry("INTEGRATION_LIST")
	PermissionIntegrationSave   = WorkspacePermissions.Entry("INTEGRATION_SAVE")

	PermissionLandingZoneDelete = WorkspacePermissions.Entry("LANDINGZONE_DELETE")
	PermissionLandingZoneList   = WorkspacePermissions.Entry("LANDINGZONE_LIST")
	PermissionLandingZoneSave   = WorkspacePermissions.Entry("LANDINGZONE_SAVE")

	PermissionManagedBuildingBlockRunSourceSave = WorkspacePermissions.Entry("MANAGED_BUILDINGBLOCKRUNSOURCE_SAVE")
	PermissionManagedBuildingBlockRunList       = WorkspacePermissions.Entry("MANAGED_BUILDINGBLOCKRUN_LIST")
	PermissionManagedBuildingBlockRunSave       = WorkspacePermissions.Entry("MANAGED_BUILDINGBLOCKRUN_SAVE")
	PermissionManagedBuildingBlockList          = WorkspacePermissions.Entry("MANAGED_BUILDINGBLOCK_LIST")
	PermissionManagedTenantImport               = WorkspacePermissions.Entry("MANAGED_TENANT_IMPORT")

	PermissionPaymentMethodList = WorkspacePermissions.Entry("PAYMENTMETHOD_LIST")

	PermissionPlatformInstanceDelete = WorkspacePermissions.Entry("PLATFORMINSTANCE_DELETE")
	PermissionPlatformInstanceList   = WorkspacePermissions.Entry("PLATFORMINSTANCE_LIST")
	PermissionPlatformInstanceSave   = WorkspacePermissions.Entry("PLATFORMINSTANCE_SAVE")

	PermissionProjectPrincipalRoleDelete = WorkspacePermissions.Entry("PROJECTPRINCIPALROLE_DELETE")
	PermissionProjectPrincipalRoleList   = WorkspacePermissions.Entry("PROJECTPRINCIPALROLE_LIST")
	PermissionProjectPrincipalRoleSave   = WorkspacePermissions.Entry("PROJECTPRINCIPALROLE_SAVE")

	PermissionProjectDelete = WorkspacePermissions.Entry("PROJECT_DELETE")
	PermissionProjectList   = WorkspacePermissions.Entry("PROJECT_LIST")
	PermissionProjectSave   = WorkspacePermissions.Entry("PROJECT_SAVE")

	PermissionServiceInstanceDelete = WorkspacePermissions.Entry("SERVICEINSTANCE_DELETE")
	PermissionServiceInstanceList   = WorkspacePermissions.Entry("SERVICEINSTANCE_LIST")
	PermissionServiceInstanceSave   = WorkspacePermissions.Entry("SERVICEINSTANCE_SAVE")

	PermissionTenantDelete = WorkspacePermissions.Entry("TENANT_DELETE")
	PermissionTenantList   = WorkspacePermissions.Entry("TENANT_LIST")
	PermissionTenantSave   = WorkspacePermissions.Entry("TENANT_SAVE")

	PermissionTfStateDelete = WorkspacePermissions.Entry("TFSTATE_DELETE")
	PermissionTfStateList   = WorkspacePermissions.Entry("TFSTATE_LIST")
	PermissionTfStateSave   = WorkspacePermissions.Entry("TFSTATE_SAVE")

	PermissionWorkspacePrincipalBindingDelete = WorkspacePermissions.Entry("WORKSPACEPRINCIPALBINDING_DELETE")
	PermissionWorkspacePrincipalBindingList   = WorkspacePermissions.Entry("WORKSPACEPRINCIPALBINDING_LIST")
	PermissionWorkspacePrincipalBindingSave   = WorkspacePermissions.Entry("WORKSPACEPRINCIPALBINDING_SAVE")

	PermissionWorkspaceUserGroupList = WorkspacePermissions.Entry("WORKSPACEUSERGROUP_LIST")

	PermissionWorkspaceDelete = WorkspacePermissions.Entry("WORKSPACE_DELETE")
	PermissionWorkspaceList   = WorkspacePermissions.Entry("WORKSPACE_LIST")
	PermissionWorkspaceSave   = WorkspacePermissions.Entry("WORKSPACE_SAVE")
)
