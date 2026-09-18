package provider

import "encoding/json/v2"

// wireCompatibility keeps json/v2's output in the shape encoding/json v1 produced, which is what
// meshStack and every Terraform state in the field have seen: the backend reads a nil collection
// sent as null differently from one sent as [] or {}, and a member order left to chance reads as
// drift on the next plan, because a JSON string this provider writes into an attribute becomes
// state and feeds the content hash of a building block definition version.
var wireCompatibility = json.JoinOptions(
	json.Deterministic(true),
	json.FormatNilSliceAsNull(true),
	json.FormatNilMapAsNull(true),
)
