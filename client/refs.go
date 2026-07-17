package client

// NamedRef is the client-side DTO for a meshObject reference that identifies its
// target by name. It is the counterpart to the meshRefByName schema builder in
// internal/provider (schema_utils.go): every {name, kind} reference block on the
// wire deserializes into this struct. Refs that carry extra fields embed it.
type NamedRef struct {
	Name string `json:"name" tfsdk:"name"`
	Kind string `json:"kind" tfsdk:"kind"`
}

// UuidRef is the client-side DTO for a meshObject reference that identifies its
// target by uuid. It is the counterpart to the meshRefByUuid schema builder in
// internal/provider (schema_utils.go): every {uuid, kind} reference block on the
// wire deserializes into this struct. Refs that carry extra fields embed it.
type UuidRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}
