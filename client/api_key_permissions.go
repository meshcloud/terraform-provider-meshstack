package client

import "strings"

// API Key Permissions aligned with Kotlin ApiKeyRightMetadataRegistry.
// See https://docs.meshcloud.io/api/authentication/api-permissions/

// ApiPermission is a permission shortcode string used for JSON serialization.
type ApiPermission string

// ApiKeyPermissions is a 3D structure:
//   - outer: groups (e.g. "Building Blocks", "Projects")
//   - middle: suffix groups within a group (e.g. DELETE, LIST, SAVE variants together)
//   - inner: scope variants (e.g. [TENANT_DELETE, ADM_TENANT_DELETE])
//
// Each permission is listed exactly as it appears in the API, no prefix derivation.
type ApiKeyPermissions [][][]ApiPermission

// AllCodes returns all valid API key permission shortcodes (flattened).
func (p ApiKeyPermissions) AllCodes() []string {
	var codes []string
	for _, group := range p {
		for _, suffixGroup := range group {
			for _, code := range suffixGroup {
				codes = append(codes, string(code))
			}
		}
	}
	return codes
}

// WorkspaceCodes returns only non-ADM_ permission shortcodes (workspace + platform builder scoped).
func (p ApiKeyPermissions) WorkspaceCodes() []string {
	var codes []string
	for _, group := range p {
		for _, suffixGroup := range group {
			for _, code := range suffixGroup {
				if !strings.HasPrefix(string(code), "ADM_") {
					codes = append(codes, string(code))
				}
			}
		}
	}
	return codes
}

// MarkdownString returns an unordered markdown list of all permissions grouped by resource.
// Each bullet shows workspace codes, then MANAGED_ codes, then ADM_ codes separated by " and ".
func (p ApiKeyPermissions) MarkdownString() string {
	var lines []string
	for _, group := range p {
		var workspace, managed, admin []string
		for _, suffixGroup := range group {
			for _, code := range suffixGroup {
				s := string(code)
				switch {
				case strings.HasPrefix(s, "ADM_"):
					admin = append(admin, "`"+s+"`")
				case strings.HasPrefix(s, "MANAGED_"):
					managed = append(managed, "`"+s+"`")
				default:
					workspace = append(workspace, "`"+s+"`")
				}
			}
		}

		var parts []string
		if len(workspace) > 0 {
			parts = append(parts, strings.Join(workspace, "/"))
		}
		if len(managed) > 0 {
			parts = append(parts, strings.Join(managed, "/"))
		}
		if len(admin) > 0 {
			parts = append(parts, strings.Join(admin, "/"))
		}
		lines = append(lines, "  - "+strings.Join(parts, " and "))
	}
	return "\n" + strings.Join(lines, "\n") + "\n"
}

// Permissions is the complete registry of API key permissions,
// aligned 1:1 with the Kotlin ApiKeyRightMetadataRegistry.
var Permissions = ApiKeyPermissions{
	// API Keys
	{
		{"APIKEY_DELETE", "ADM_APIKEY_DELETE"},
		{"APIKEY_LIST", "ADM_APIKEY_LIST"},
		{"APIKEY_SAVE", "ADM_APIKEY_SAVE"},
	},
	// Building Blocks
	{
		{"BUILDINGBLOCK_DELETE", "ADM_BUILDINGBLOCK_DELETE"},
		{"BUILDINGBLOCK_LIST", "ADM_BUILDINGBLOCK_LIST", "MANAGED_BUILDINGBLOCK_LIST"},
		{"BUILDINGBLOCK_SAVE", "ADM_BUILDINGBLOCK_SAVE", "MANAGED_BUILDINGBLOCK_SAVE"},
	},
	// Building Block Definitions
	{
		{"BUILDINGBLOCKDEFINITION_DELETE", "ADM_BUILDINGBLOCKDEFINITION_DELETE"},
		{"BUILDINGBLOCKDEFINITION_LIST", "ADM_BUILDINGBLOCKDEFINITION_LIST"},
		{"BUILDINGBLOCKDEFINITION_SAVE", "ADM_BUILDINGBLOCKDEFINITION_SAVE"},
		{"ADM_REVIEW_PUBLICATION"},
	},
	// Building Block Runs
	{
		{"MANAGED_BUILDINGBLOCKRUN_LIST", "ADM_BUILDINGBLOCKRUN_LIST"},
		{"MANAGED_BUILDINGBLOCKRUN_SAVE", "ADM_BUILDINGBLOCKRUN_SAVE"},
		{"MANAGED_BUILDINGBLOCKRUNSOURCE_SAVE", "ADM_BUILDINGBLOCKRUNSOURCE_SAVE"},
	},
	// Building Block Runners
	{
		{"BUILDINGBLOCKRUNNER_DELETE", "ADM_BUILDINGBLOCKRUNNER_DELETE"},
		{"BUILDINGBLOCKRUNNER_LIST", "ADM_BUILDINGBLOCKRUNNER_LIST"},
		{"BUILDINGBLOCKRUNNER_SAVE", "ADM_BUILDINGBLOCKRUNNER_SAVE"},
	},
	// Communication Definitions
	{
		{"COMMUNICATIONDEFINITION_DELETE", "ADM_COMMUNICATIONDEFINITION_DELETE"},
		{"COMMUNICATIONDEFINITION_LIST", "ADM_COMMUNICATIONDEFINITION_LIST"},
		{"COMMUNICATIONDEFINITION_SAVE", "ADM_COMMUNICATIONDEFINITION_SAVE"},
	},
	// Communications
	{
		{"COMMUNICATION_DELETE", "ADM_COMMUNICATION_DELETE"},
		{"COMMUNICATION_LIST", "ADM_COMMUNICATION_LIST"},
		{"COMMUNICATION_SAVE", "ADM_COMMUNICATION_SAVE"},
	},
	// Event Logs
	{
		{"EVENTLOG_LIST", "ADM_EVENTLOG_LIST"},
	},
	// Integrations
	{
		{"INTEGRATION_DELETE", "ADM_INTEGRATION_DELETE"},
		{"INTEGRATION_LIST", "ADM_INTEGRATION_LIST"},
		{"INTEGRATION_SAVE", "ADM_INTEGRATION_SAVE"},
	},
	// Landing Zones
	{
		{"LANDINGZONE_DELETE", "ADM_LANDINGZONE_DELETE"},
		{"LANDINGZONE_LIST", "ADM_LANDINGZONE_LIST"},
		{"LANDINGZONE_SAVE", "ADM_LANDINGZONE_SAVE"},
	},
	// Payment Methods
	{
		{"ADM_PAYMENTMETHOD_DELETE"},
		{"PAYMENTMETHOD_LIST", "ADM_PAYMENTMETHOD_LIST"},
		{"ADM_PAYMENTMETHOD_SAVE"},
	},
	// Platform Instances, Platform Types, Locations
	{
		{"PLATFORMINSTANCE_DELETE", "ADM_PLATFORMINSTANCE_DELETE"},
		{"PLATFORMINSTANCE_LIST", "ADM_PLATFORMINSTANCE_LIST"},
		{"PLATFORMINSTANCE_SAVE", "ADM_PLATFORMINSTANCE_SAVE"},
	},
	// Project Role Bindings
	{
		{"PROJECTPRINCIPALROLE_DELETE", "ADM_PROJECTPRINCIPALROLE_DELETE"},
		{"PROJECTPRINCIPALROLE_LIST", "ADM_PROJECTPRINCIPALROLE_LIST"},
		{"PROJECTPRINCIPALROLE_SAVE", "ADM_PROJECTPRINCIPALROLE_SAVE"},
	},
	// Project Roles
	{
		{"ADM_PROJECTROLE_DELETE"},
		{"ADM_PROJECTROLE_SAVE"},
	},
	// Projects
	{
		{"PROJECT_DELETE", "ADM_PROJECT_DELETE"},
		{"PROJECT_LIST", "ADM_PROJECT_LIST"},
		{"PROJECT_SAVE", "ADM_PROJECT_SAVE"},
	},
	// Service Instances
	{
		{"SERVICEINSTANCE_DELETE", "ADM_SERVICEINSTANCE_DELETE"},
		{"SERVICEINSTANCE_LIST", "ADM_SERVICEINSTANCE_LIST"},
		{"SERVICEINSTANCE_SAVE", "ADM_SERVICEINSTANCE_SAVE"},
	},
	// Tag Definitions
	{
		{"ADM_TAGDEFINITION_DELETE"},
		{"ADM_TAGDEFINITION_LIST"},
		{"ADM_TAGDEFINITION_SAVE"},
	},
	// Tenants
	{
		{"TENANT_DELETE", "ADM_TENANT_DELETE"},
		{"MANAGED_TENANT_IMPORT", "ADM_TENANT_IMPORT"},
		{"TENANT_LIST", "ADM_TENANT_LIST"},
		{"TENANT_SAVE", "ADM_TENANT_SAVE"},
	},
	// Terraform States
	{
		{"TFSTATE_DELETE", "ADM_TFSTATE_DELETE", "MANAGED_TFSTATE_DELETE"},
		{"TFSTATE_LIST", "ADM_TFSTATE_LIST", "MANAGED_TFSTATE_LIST"},
		{"TFSTATE_SAVE", "ADM_TFSTATE_SAVE", "MANAGED_TFSTATE_SAVE"},
	},
	// Users
	{
		{"ADM_USER_DELETE"},
		{"ADM_USER_LIST"},
		{"ADM_USER_SAVE"},
	},
	// Workspace Role Bindings
	{
		{"WORKSPACEPRINCIPALBINDING_DELETE", "ADM_WORKSPACEPRINCIPALBINDING_DELETE"},
		{"WORKSPACEPRINCIPALBINDING_LIST", "ADM_WORKSPACEPRINCIPALBINDING_LIST"},
		{"WORKSPACEPRINCIPALBINDING_SAVE", "ADM_WORKSPACEPRINCIPALBINDING_SAVE"},
	},
	// Workspace User Groups
	{
		{"WORKSPACEUSERGROUP_LIST", "ADM_WORKSPACEUSERGROUP_LIST"},
	},
	// Workspaces
	{
		{"WORKSPACE_DELETE", "ADM_WORKSPACE_DELETE"},
		{"WORKSPACE_LIST", "ADM_WORKSPACE_LIST"},
		{"WORKSPACE_SAVE", "ADM_WORKSPACE_SAVE"},
	},
}

// Convenience functions used by consumers.

// AllApiKeyPermissions returns all valid API key permission shortcodes.
func AllApiKeyPermissions() []string {
	return Permissions.AllCodes()
}

// WorkspacePermissionCodes returns only workspace-scoped permission shortcodes.
func WorkspacePermissionCodes() []string {
	return Permissions.WorkspaceCodes()
}
