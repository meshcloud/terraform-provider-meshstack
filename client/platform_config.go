package client

type MeshTagConfig struct {
	NamespacePrefix string      `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []TagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type TagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}
