package provider

type MeshProject struct {
	ApiVersion string              `json:"apiVersion" tfsdk:"api_version"`
	Kind       string              `json:"kind" tfsdk:"kind"`
	Metadata   MeshProjectMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshProjectSpec     `json:"spec" tfsdk:"spec"`
}

type MeshProjectMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshProjectSpec struct {
	DisplayName                       string              `json:"displayName" tfsdk:"display_name"`
	Tags                              map[string][]string `json:"tags" tfsdk:"tags"`
	PaymentMethodIdentifier           *string             `json:"paymentMethodIdentifier" tfsdk:"payment_method_identifier"`
	SubstitutePaymentMethodIdentifier *string             `json:"substitutePaymentMethodIdentifier" tfsdk:"substitute_payment_method_identifier"`
}

type MeshProjectCreate struct {
	Metadata MeshProjectCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshProjectSpec           `json:"spec" tfsdk:"spec"`
}

type MeshProjectCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}
