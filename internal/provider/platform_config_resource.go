package provider

type MeshPlatformSpecModel struct {
	Config       *PlatformConfigModel           `tfsdk:"config"`
	Availability *MeshPlatformAvailabilityModel `tfsdk:"availability"`
}

type PlatformConfigModel struct {
	Type       string                        `tfsdk:"type"`
	AKS        *AKSPlatformConfigModel       `tfsdk:"aks"`
	Azure      *AzurePlatformConfigModel     `tfsdk:"azure"`
	AzureRG    *AzureRGPlatformConfigModel   `tfsdk:"azurerg"`
	AWS        *AWSPlatformConfigModel       `tfsdk:"aws"`
	GCP        *GCPPlatformConfigModel       `tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig     `tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfigModel `tfsdk:"openshift"`
}

type MeshTagConfigModel struct {
	NamespacePrefix string           `tfsdk:"namespace_prefix"`
	TagMappers      []TagMapperModel `tfsdk:"tag_mappers"`
}

type TagMapperModel struct {
	Key          string `tfsdk:"key"`
	ValuePattern string `tfsdk:"value_pattern"`
}

type MeshPlatformAvailabilityModel struct {
	Restriction            string   `tfsdk:"restriction"`
	RestrictedToWorkspaces []string `tfsdk:"restricted_to_workspaces"`
	MarketplaceStatus      string   `tfsdk:"marketplace_status"`
}
