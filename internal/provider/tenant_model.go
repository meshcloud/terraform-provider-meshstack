package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// tenantRef is the computed {kind, uuid} self-reference of a meshTenant.
type tenantRef struct {
	Kind string `tfsdk:"kind"`
	Uuid string `tfsdk:"uuid"`
}

// tenantModel wraps a MeshTenant DTO with its computed ref. It backs the plural meshstack_tenants
// data source, whose element schema (ref + metadata + spec + status) matches the DTO exactly. The
// resource needs the extra provider-only wait_for_completion toggle and uses tenantResourceModel.
type tenantModel struct {
	*client.MeshTenant
	Ref tenantRef `tfsdk:"ref"`
}

func tenantModelFromDto(tenant *client.MeshTenant) tenantModel {
	return tenantModel{
		MeshTenant: tenant,
		Ref:        tenantRef{Kind: client.MeshObjectKind.Tenant, Uuid: tenant.Metadata.Uuid},
	}
}

// tenantResourceModel backs the unsuffixed meshstack_tenant resource. It reuses the DTO sub-types
// (metadata/spec/status) and adds the provider-only wait_for_completion toggle, which is not part of
// the API DTO.
type tenantResourceModel struct {
	Ref               tenantRef                 `tfsdk:"ref"`
	Metadata          client.MeshTenantMetadata `tfsdk:"metadata"`
	Spec              client.MeshTenantSpec     `tfsdk:"spec"`
	Status            client.MeshTenantStatus   `tfsdk:"status"`
	WaitForCompletion bool                      `tfsdk:"wait_for_completion"`
}

// tenantResourceModelFromDto builds the resource state from an API response. specQuotas is the known
// (configured) spec.quotas carried from plan/state: spec.quotas is Optional (not computed), so it must
// echo the configured value verbatim to avoid an inconsistent-result-after-apply error when the
// backend defaults or reorders quotas.
func tenantResourceModelFromDto(dto *client.MeshTenant, specQuotas clientTypes.Set[client.MeshTenantQuota], waitForCompletion bool) tenantResourceModel {
	spec := dto.Spec
	spec.Quotas = specQuotas
	return tenantResourceModel{
		Ref:               tenantRef{Kind: client.MeshObjectKind.Tenant, Uuid: dto.Metadata.Uuid},
		Metadata:          dto.Metadata,
		Spec:              spec,
		Status:            dto.Status,
		WaitForCompletion: waitForCompletion,
	}
}

// tenantConverterOptions renders the generic Set-typed quota slices (client/types.Set) as Terraform
// sets rather than lists, matching the SetNestedAttribute schema for tenant quotas.
func tenantConverterOptions() generic.ConverterOptions {
	return generic.ConverterOptions{generic.WithSliceTypeAsSet(clientTypes.IsSet)}
}
