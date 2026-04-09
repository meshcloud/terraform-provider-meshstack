package client

// meshObjectKind provides typed constants for meshObject kind strings used across the provider.
type meshObjectKind struct {
	BuildingBlock                  string
	BuildingBlockDefinition        string
	BuildingBlockDefinitionVersion string
	BuildingBlockRunner            string
	Integration                    string
	LandingZone                    string
	Location                       string
	PaymentMethod                  string
	Platform                       string
	PlatformType                   string
	Project                        string
	ProjectGroupBinding            string
	ProjectRole                    string
	ProjectUserBinding             string
	ServiceInstance                string
	TagDefinition                  string
	Tenant                         string
	Workspace                      string
	WorkspaceGroupBinding          string
	WorkspaceUserBinding           string
}

var MeshObjectKind = meshObjectKind{
	BuildingBlock:                  "meshBuildingBlock",
	BuildingBlockDefinition:        "meshBuildingBlockDefinition",
	BuildingBlockDefinitionVersion: "meshBuildingBlockDefinitionVersion",
	BuildingBlockRunner:            "meshBuildingBlockRunner",
	Integration:                    "meshIntegration",
	LandingZone:                    "meshLandingZone",
	Location:                       "meshLocation",
	PaymentMethod:                  "meshPaymentMethod",
	Platform:                       "meshPlatform",
	PlatformType:                   "meshPlatformType",
	Project:                        "meshProject",
	ProjectGroupBinding:            "meshProjectGroupBinding",
	ProjectRole:                    "meshProjectRole",
	ProjectUserBinding:             "meshProjectUserBinding",
	ServiceInstance:                "meshServiceInstance",
	TagDefinition:                  "meshTagDefinition",
	Tenant:                         "meshTenant",
	Workspace:                      "meshWorkspace",
	WorkspaceGroupBinding:          "meshWorkspaceGroupBinding",
	WorkspaceUserBinding:           "meshWorkspaceUserBinding",
}
