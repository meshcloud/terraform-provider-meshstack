package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	apiPermissionRegex          = regexp.MustCompile(`^(MANAGED_|ADM_)?[A-Z]+_[A-Z]+$`)
	workspaceApiPermissionRegex = regexp.MustCompile(`^(MANAGED_)?[A-Z]+_[A-Z]+$`)
)

func apiPermissionValidator() validator.String {
	return stringvalidator.RegexMatches(apiPermissionRegex,
		"must be a permission such as TENANT_LIST, MANAGED_TENANT_LIST or ADM_TENANT_LIST: "+
			"an optional MANAGED_ or ADM_ prefix, a resource and an action, in upper-case letters")
}

func workspaceApiPermissionValidator() validator.String {
	return stringvalidator.RegexMatches(workspaceApiPermissionRegex,
		"must be a workspace permission such as TENANT_LIST or MANAGED_TENANT_LIST: "+
			"an optional MANAGED_ prefix, a resource and an action, in upper-case letters. ADM_ permissions are not allowed here")
}
