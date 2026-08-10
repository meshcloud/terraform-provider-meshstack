package provider

import (
	"context"
	"maps"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// tenantSpecV1 mirrors client.MeshTenantSpec as it stood while the deprecated list-form spec.quotas
// still existed. Keep it a standalone struct: reading prior state through the current spec is the bug
// that broke the landing zone's 0 -> 1 upgrade.
type tenantSpecV1 struct {
	PlatformRef      client.UuidRef                          `tfsdk:"platform_ref"`
	PlatformTenantId *string                                 `tfsdk:"platform_tenant_id"`
	LandingZoneRef   *client.NamedRef                        `tfsdk:"landing_zone_ref"`
	RequestedQuotas  map[string]client.RequestQuotaValue     `tfsdk:"requested_quotas"`
	Quotas           clientTypes.Set[client.MeshTenantQuota] `tfsdk:"quotas"`
}

type tenantResourceModelV1 struct {
	Ref               tenantRef                 `tfsdk:"ref"`
	Metadata          client.MeshTenantMetadata `tfsdk:"metadata"`
	Spec              tenantSpecV1              `tfsdk:"spec"`
	Status            client.MeshTenantStatus   `tfsdk:"status"`
	WaitForCompletion bool                      `tfsdk:"wait_for_completion"`
}

// tenantSchemaV1Once builds the schema version 1 shape for the state upgrader: the current schema with
// the removed spec.quotas attribute added back.
var tenantSchemaV1Once = sync.OnceValue(func() schema.Schema {
	var schemaResp resource.SchemaResponse
	(&tenantResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	s.Version = 1

	spec, ok := s.Attributes["spec"].(schema.SingleNestedAttribute)
	if !ok {
		panic("tenant spec attribute is not a SingleNestedAttribute")
	}
	spec.Attributes = maps.Clone(spec.Attributes)
	spec.Attributes["quotas"] = schema.SetNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key":   schema.StringAttribute{Required: true},
				"value": schema.Int64Attribute{Required: true},
			},
		},
	}
	s.Attributes = maps.Clone(s.Attributes)
	s.Attributes["spec"] = spec

	return s
})

// upgradeFromV1 migrates state written while spec.quotas existed. State that already carries
// spec.requested_quotas keeps it, because v0.24.3 accepted both fields only when they described the
// same quotas.
func (r *tenantResource) upgradeFromV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	prior := generic.Get[tenantResourceModelV1](ctx, req.State, &resp.Diagnostics, tenantConverterOptions()...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestedQuotas := prior.Spec.RequestedQuotas
	if len(requestedQuotas) == 0 {
		requestedQuotas = quotaListToRequestedMap(prior.Spec.Quotas)
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, tenantResourceModel{
		Ref:      prior.Ref,
		Metadata: prior.Metadata,
		Spec: client.MeshTenantSpec{
			PlatformRef:      prior.Spec.PlatformRef,
			PlatformTenantId: prior.Spec.PlatformTenantId,
			LandingZoneRef:   prior.Spec.LandingZoneRef,
			RequestedQuotas:  requestedQuotas,
		},
		Status:            prior.Status,
		WaitForCompletion: prior.WaitForCompletion,
	}, tenantConverterOptions()...)...)
}
