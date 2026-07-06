package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/hash"
)

const (
	errorHashVersion   = "err"
	currentHashVersion = "v2"
)

// represents a content hash of a building block definition version, which is used to detect changes in the version_spec.
// because the version_spec is subject to change w.r.t. new fields etc., the hash is versioned
// this means we can only meaningfully compare hashes of the same version
// use func indicatesChangesTowardsHash(hashStr string) to detect whether an instance of BuildingBlockDefinitionVersionContentHash
// differs from a previously calculated hash string representation.
type BuildingBlockDefinitionVersionContentHash struct {
	hashVersion string
	hashValue   string
}

func calculateBuildingBlockDefinitionVersionContentHash(versionSpecDto client.MeshBuildingBlockDefinitionVersionSpec, diags *diag.Diagnostics) BuildingBlockDefinitionVersionContentHash {
	if result, err := func() (string, error) {
		// Safeguard against accidentally hashing plaintext secret values (the backend only ever returns
		// hashes, so plaintext would make the hash unstable). Detection is on the typed DTO so user data -
		// e.g. an input named "plaintext" or a STATIC argument whose JSON carries a "plaintext" key - is
		// never mistaken for a secret. Callers leave the hash unknown when a secret rotates (issue #196).
		if versionSpecContainsPlaintextSecret(versionSpecDto) {
			return "", errors.New("version_spec carries a plaintext secret value, which must not be hashed")
		}

		// Ignore version, state, and buildingBlockDefinitionRef fields by setting them to constant values, always!
		versionSpecDto.VersionNumber = nil
		versionSpecDto.State = nil
		versionSpecDto.BuildingBlockDefinitionRef = nil

		// Converting it first from/to JSON makes hashing more stable, as fields with 'omitempty' are ignored.
		// Additionally, all numbers are converted to float64, even integers (which also allows changing DTO model types later on).
		// Also, the current Hasher implementation does not support structs for now, but map[string]any works!
		var buffer bytes.Buffer
		if err := json.NewEncoder(&buffer).Encode(versionSpecDto); err != nil {
			return "", err
		}
		var converted any
		if err := json.NewDecoder(&buffer).Decode(&converted); err != nil {
			return "", err
		}

		versionSpecHash, err := hash.Hasher{}.Hash(converted)
		if err != nil {
			return "", err
		}

		// add some versioning prefix to migrate possible changes in the hashes later on,
		// but let's hope migrating/fixing the hashes is never required
		return versionSpecHash.Hex(), nil

	}(); err != nil {
		diags.AddError("Failed to determine content hash", fmt.Sprintf(
			"Content hashing of version_spec as client DTO failed: %s", err.Error(),
		))

		return errorHash()
	} else {

		return BuildingBlockDefinitionVersionContentHash{
			hashVersion: currentHashVersion,
			hashValue:   result,
		}
	}
}

// we can only safely know that two definition versions differ in case they have both the same hash version
// and different hash values. If the hash versions differ, we cannot know whether the content has changed or not.
func (h BuildingBlockDefinitionVersionContentHash) indicatesChangesTowardsHash(hashStr string) bool {

	getVersionedHashFromString := func(encoded string) BuildingBlockDefinitionVersionContentHash {
		// v1 hashes were just simple strings in the format of "v1:<hash>"
		// later version representations are all base64 encoded strings
		if strings.HasPrefix(encoded, "v1:") { // safe check, as base64 cannot contain ":"
			return BuildingBlockDefinitionVersionContentHash{
				hashVersion: "v1",
				hashValue:   encoded,
			}
		} else {
			h := BuildingBlockDefinitionVersionContentHash{}
			h.loadFromBase64(encoded)
			return h
		}
	}

	other := getVersionedHashFromString(hashStr)

	return (h.hashVersion == other.hashVersion) && (h.hashValue != other.hashValue)
}

func (h BuildingBlockDefinitionVersionContentHash) toBase64() string {
	text := h.hashVersion + ":" + h.hashValue
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	return encoded
}

func (h *BuildingBlockDefinitionVersionContentHash) loadFromBase64(encoded string) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		h.hashVersion = errorHashVersion
		return
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		h.hashVersion = errorHashVersion
		return
	}

	h.hashVersion = parts[0]
	h.hashValue = parts[1]
}

func errorHash() BuildingBlockDefinitionVersionContentHash {
	return BuildingBlockDefinitionVersionContentHash{
		hashVersion: errorHashVersion,
		hashValue:   "",
	}
}
