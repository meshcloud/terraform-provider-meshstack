package provider

// tenantV4Ref is the computed `ref` output of the deprecated meshstack_tenant_v4 resource.
type tenantV4Ref struct {
	Kind string `tfsdk:"kind"`
	Uuid string `tfsdk:"uuid"`
}
