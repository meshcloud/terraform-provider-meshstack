package provider

import (
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
)

func TestApiPermissionRegex(t *testing.T) {
	for _, code := range client.Permissions.AllCodes() {
		assert.Regexp(t, apiPermissionRegex, code)
	}
	for _, code := range []string{"BUILDINGBLOCKRUN_LIST", "MANAGED_FOO_BAR", "ADM_FOO_BAR"} {
		assert.Regexp(t, apiPermissionRegex, code)
	}
	for _, code := range []string{"", "TENANT", "TENANT_", "_LIST", "tenant_list", "Tenant_List", "TENANT_LIST ", "TENANT-LIST", "TENANT_LIST_ALL", "ADM_MANAGED_TENANT_LIST", "MANAGED_ADM_TENANT_LIST", "ADMIN_TENANT_LIST", "TENANT2_LIST", "*"} {
		assert.NotRegexp(t, apiPermissionRegex, code)
	}
}

func TestWorkspaceApiPermissionRegex(t *testing.T) {
	for _, code := range client.WorkspacePermissionCodes() {
		assert.Regexp(t, workspaceApiPermissionRegex, code)
	}
	for _, code := range []string{"BUILDINGBLOCKRUN_LIST", "MANAGED_FOO_BAR"} {
		assert.Regexp(t, workspaceApiPermissionRegex, code)
	}
	for _, code := range []string{"", "TENANT", "tenant_list", "TENANT_LIST_ALL", "ADM_TENANT_LIST", "ADM_FOO_BAR", "MANAGED_ADM_TENANT_LIST"} {
		assert.NotRegexp(t, workspaceApiPermissionRegex, code)
	}
}
