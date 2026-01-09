package client

type MeshLocation struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshLocationStatus   `json:"status" tfsdk:"status"`
}

type MeshLocationMetadata struct {
	Name string `json:"name" tfsdk:"name"`
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshLocationSpec struct {
	DisplayName string `json:"displayName" tfsdk:"display_name"`
	Description string `json:"description" tfsdk:"description"`
}

type MeshLocationStatus struct {
	IsPublic bool `json:"isPublic" tfsdk:"is_public"`
}

type MeshLocationCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec           `json:"spec" tfsdk:"spec"`
}

type MeshLocationCreateMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshLocationClient struct {
	meshObjectClient[MeshLocation]
}

func (c MeshLocationClient) Read(name string) (*MeshLocation, error) {
	return c.get(name)
}

func (c MeshLocationClient) Create(location *MeshLocationCreate) (*MeshLocation, error) {
	return c.post(location)
}

func (c MeshLocationClient) Update(name string, location *MeshLocationCreate) (*MeshLocation, error) {
	return c.put(name, location)
}

func (c MeshLocationClient) Delete(name string) error {
	return c.delete(name)
}
