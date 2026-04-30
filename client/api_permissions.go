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
	PermissionManagedTfStateDelete              = WorkspacePermissions.Entry("MANAGED_TFSTATE_DELETE")
	PermissionManagedTfStateList                = WorkspacePermissions.Entry("MANAGED_TFSTATE_LIST")
	PermissionManagedTfStateSave                = WorkspacePermissions.Entry("MANAGED_TFSTATE_SAVE")

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

// Admin Permissions (cross-workspace).
var (
	AdminPermissions = enum.Enum[ApiPermission]{}

	PermissionAdmBuildingBlockDefinitionDelete = AdminPermissions.Entry("ADM_BUILDINGBLOCKDEFINITION_DELETE")
	PermissionAdmBuildingBlockDefinitionList   = AdminPermissions.Entry("ADM_BUILDINGBLOCKDEFINITION_LIST")
	PermissionAdmBuildingBlockDefinitionSave   = AdminPermissions.Entry("ADM_BUILDINGBLOCKDEFINITION_SAVE")

	PermissionAdmBuildingBlockRunnerDelete = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUNNER_DELETE")
	PermissionAdmBuildingBlockRunnerList   = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUNNER_LIST")
	PermissionAdmBuildingBlockRunnerSave   = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUNNER_SAVE")

	PermissionAdmBuildingBlockRunList       = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUN_LIST")
	PermissionAdmBuildingBlockRunSave       = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUN_SAVE")
	PermissionAdmBuildingBlockRunSourceSave = AdminPermissions.Entry("ADM_BUILDINGBLOCKRUNSOURCE_SAVE")

	PermissionAdmBuildingBlockDelete = AdminPermissions.Entry("ADM_BUILDINGBLOCK_DELETE")
	PermissionAdmBuildingBlockList   = AdminPermissions.Entry("ADM_BUILDINGBLOCK_LIST")
	PermissionAdmBuildingBlockSave   = AdminPermissions.Entry("ADM_BUILDINGBLOCK_SAVE")

	PermissionAdmCommunicationDefinitionDelete = AdminPermissions.Entry("ADM_COMMUNICATIONDEFINITION_DELETE")
	PermissionAdmCommunicationDefinitionList   = AdminPermissions.Entry("ADM_COMMUNICATIONDEFINITION_LIST")
	PermissionAdmCommunicationDefinitionSave   = AdminPermissions.Entry("ADM_COMMUNICATIONDEFINITION_SAVE")

	PermissionAdmCommunicationDelete = AdminPermissions.Entry("ADM_COMMUNICATION_DELETE")
	PermissionAdmCommunicationList   = AdminPermissions.Entry("ADM_COMMUNICATION_LIST")
	PermissionAdmCommunicationSave   = AdminPermissions.Entry("ADM_COMMUNICATION_SAVE")

	PermissionAdmEventLogList = AdminPermissions.Entry("ADM_EVENTLOG_LIST")

	PermissionAdmIntegrationDelete = AdminPermissions.Entry("ADM_INTEGRATION_DELETE")
	PermissionAdmIntegrationList   = AdminPermissions.Entry("ADM_INTEGRATION_LIST")
	PermissionAdmIntegrationSave   = AdminPermissions.Entry("ADM_INTEGRATION_SAVE")

	PermissionAdmLandingZoneDelete = AdminPermissions.Entry("ADM_LANDINGZONE_DELETE")
	PermissionAdmLandingZoneList   = AdminPermissions.Entry("ADM_LANDINGZONE_LIST")
	PermissionAdmLandingZoneSave   = AdminPermissions.Entry("ADM_LANDINGZONE_SAVE")

	PermissionAdmPaymentMethodDelete = AdminPermissions.Entry("ADM_PAYMENTMETHOD_DELETE")
	PermissionAdmPaymentMethodList   = AdminPermissions.Entry("ADM_PAYMENTMETHOD_LIST")
	PermissionAdmPaymentMethodSave   = AdminPermissions.Entry("ADM_PAYMENTMETHOD_SAVE")

	PermissionAdmPlatformInstanceDelete = AdminPermissions.Entry("ADM_PLATFORMINSTANCE_DELETE")
	PermissionAdmPlatformInstanceList   = AdminPermissions.Entry("ADM_PLATFORMINSTANCE_LIST")
	PermissionAdmPlatformInstanceSave   = AdminPermissions.Entry("ADM_PLATFORMINSTANCE_SAVE")

	PermissionAdmProjectPrincipalRoleDelete = AdminPermissions.Entry("ADM_PROJECTPRINCIPALROLE_DELETE")
	PermissionAdmProjectPrincipalRoleList   = AdminPermissions.Entry("ADM_PROJECTPRINCIPALROLE_LIST")
	PermissionAdmProjectPrincipalRoleSave   = AdminPermissions.Entry("ADM_PROJECTPRINCIPALROLE_SAVE")

	PermissionAdmProjectRoleDelete = AdminPermissions.Entry("ADM_PROJECTROLE_DELETE")
	PermissionAdmProjectRoleSave   = AdminPermissions.Entry("ADM_PROJECTROLE_SAVE")

	PermissionAdmProjectDelete = AdminPermissions.Entry("ADM_PROJECT_DELETE")
	PermissionAdmProjectList   = AdminPermissions.Entry("ADM_PROJECT_LIST")
	PermissionAdmProjectSave   = AdminPermissions.Entry("ADM_PROJECT_SAVE")

	PermissionAdmReviewPublication = AdminPermissions.Entry("ADM_REVIEW_PUBLICATION")

	PermissionAdmServiceInstanceDelete = AdminPermissions.Entry("ADM_SERVICEINSTANCE_DELETE")
	PermissionAdmServiceInstanceList   = AdminPermissions.Entry("ADM_SERVICEINSTANCE_LIST")
	PermissionAdmServiceInstanceSave   = AdminPermissions.Entry("ADM_SERVICEINSTANCE_SAVE")

	PermissionAdmTagDefinitionDelete = AdminPermissions.Entry("ADM_TAGDEFINITION_DELETE")
	PermissionAdmTagDefinitionList   = AdminPermissions.Entry("ADM_TAGDEFINITION_LIST")
	PermissionAdmTagDefinitionSave   = AdminPermissions.Entry("ADM_TAGDEFINITION_SAVE")

	PermissionAdmTenantDelete = AdminPermissions.Entry("ADM_TENANT_DELETE")
	PermissionAdmTenantList   = AdminPermissions.Entry("ADM_TENANT_LIST")
	PermissionAdmTenantSave   = AdminPermissions.Entry("ADM_TENANT_SAVE")

	PermissionAdmTfStateDelete = AdminPermissions.Entry("ADM_TFSTATE_DELETE")
	PermissionAdmTfStateList   = AdminPermissions.Entry("ADM_TFSTATE_LIST")
	PermissionAdmTfStateSave   = AdminPermissions.Entry("ADM_TFSTATE_SAVE")

	PermissionAdmUserDelete = AdminPermissions.Entry("ADM_USER_DELETE")
	PermissionAdmUserList   = AdminPermissions.Entry("ADM_USER_LIST")
	PermissionAdmUserSave   = AdminPermissions.Entry("ADM_USER_SAVE")

	PermissionAdmWorkspaceDelete = AdminPermissions.Entry("ADM_WORKSPACE_DELETE")
	PermissionAdmWorkspaceList   = AdminPermissions.Entry("ADM_WORKSPACE_LIST")
	PermissionAdmWorkspaceSave   = AdminPermissions.Entry("ADM_WORKSPACE_SAVE")

	PermissionAdmWorkspacePrincipalBindingDelete = AdminPermissions.Entry("ADM_WORKSPACEPRINCIPALBINDING_DELETE")
	PermissionAdmWorkspacePrincipalBindingList   = AdminPermissions.Entry("ADM_WORKSPACEPRINCIPALBINDING_LIST")
	PermissionAdmWorkspacePrincipalBindingSave   = AdminPermissions.Entry("ADM_WORKSPACEPRINCIPALBINDING_SAVE")

	PermissionAdmWorkspaceUserGroupList = AdminPermissions.Entry("ADM_WORKSPACEUSERGROUP_LIST")
)

// AllApiKeyPermissions returns all valid API key permission shortcodes.
func AllApiKeyPermissions() []string {
	combined := append(WorkspacePermissions.Strings(), AdminPermissions.Strings()...)
	return combined
}
