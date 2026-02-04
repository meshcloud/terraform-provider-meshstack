package client

type CustomPlatformConfig struct {
	PlatformTypeRef PlatformTypeRef       `json:"platformTypeRef" tfsdk:"platform_type_ref"`
	Metering        *CustomMeteringConfig `json:"metering,omitempty" tfsdk:"metering"`
}

type PlatformTypeRef struct {
	Name string `json:"name" tfsdk:"name"`
	Kind string `json:"kind" tfsdk:"kind"`
}

type CustomMeteringConfig struct {
	Processing *MeshPlatformMeteringProcessingConfig `json:"processing,omitempty" tfsdk:"processing"`
}
