package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/hash"
)

// currentHashVersion tags every hash this provider computes. Bump it whenever the hashing algorithm changes
// (any change that makes a given version_spec produce a different digest, e.g. folding display_order back in).
// The bump is what keeps such a change safe: a hash stored by an older provider then compares as a *different
// version* and is treated as incomparable — the caller recomputes the stored spec at the current version
// instead of reporting a spurious "changed", which would rerun every already-released building block. Changing
// the algorithm without bumping this is a silent breaking change.
const (
	// v3: display_order folded into the hash. v4: dependencies hashed as dependencyDefinitionRefs.
	// v5: manual building block outputs are hashed as the tracked-override subset (sparse), not the full
	// backend-derived one-per-input set, matching the sparse config/state model.
	currentHashVersion = 5
)

// represents a content hash of a building block definition version, which is used to detect changes in the version_spec.
// because the version_spec is subject to change w.r.t. new fields etc., the hash is versioned.
type BuildingBlockDefinitionVersionContentHash struct {
	hashVersion int
	hashValue   string
}

// hashComparison is the three-way result of comparing a freshly computed content hash against a
// previously stored one. Two hash *values* are only meaningful to compare when they were produced
// by the same hash version (algorithm); across versions the values are unrelated, so the outcome is
// hashIncomparable.
type hashComparison int

const (
	hashIncomparable hashComparison = iota
	hashSame
	hashDifferent
)

func calculateBuildingBlockDefinitionVersionContentHash(versionSpecDto client.MeshBuildingBlockDefinitionVersionSpec, diags *diag.Diagnostics) BuildingBlockDefinitionVersionContentHash {
	if result, err := func() (string, error) {
		// Safeguard against accidentally hashing plaintext secret values (the backend only ever returns
		// hashes, so plaintext would make the hash unstable). Detection is on the typed DTO so user data -
		// e.g. an input named "plaintext" or a STATIC argument whose JSON carries a "plaintext" key - is
		// never mistaken for a secret. Callers leave the hash unknown when a secret rotates (issue #196).
		if versionSpecContainsPlaintextSecret(versionSpecDto) {
			return "", errors.New("version_spec carries a plaintext secret value, which must not be hashed")
		}

		// Hash the config-shaped outputs. For manual building blocks only the tracked overrides are part of
		// config/state (the rest are backend-derived), so prune to that subset. This is idempotent on an
		// already-sparse spec, so the plan-side hash (sparse) and the read-back hash (full response) agree.
		versionSpecDto.Outputs = manualTrackedOutputs(versionSpecDto)

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

		return versionSpecHash.Hex(), nil

	}(); err != nil {
		diags.AddError("Failed to determine content hash", fmt.Sprintf(
			"Content hashing of version_spec as client DTO failed: %s", err.Error(),
		))

		return BuildingBlockDefinitionVersionContentHash{}
	} else {

		return BuildingBlockDefinitionVersionContentHash{
			hashVersion: currentHashVersion,
			hashValue:   result,
		}
	}
}

func getVersionedHashFromString(encoded string) (BuildingBlockDefinitionVersionContentHash, error) {
	// v1 hashes were just simple strings in the format of "v1:<hash>"
	// later version representations are all base64 encoded strings
	if strings.HasPrefix(encoded, "v1:") { // safe check, as base64 cannot contain ":"
		return BuildingBlockDefinitionVersionContentHash{
			hashVersion: 1,
			hashValue:   encoded,
		}, nil
	} else {
		h := BuildingBlockDefinitionVersionContentHash{}
		err := h.loadFromBase64(encoded)
		return h, err
	}
}

func (h BuildingBlockDefinitionVersionContentHash) compareToStored(storedStr string) hashComparison {
	other, err := getVersionedHashFromString(storedStr)

	if err != nil || h.hashVersion != other.hashVersion {
		return hashIncomparable
	}

	if h.hashValue == other.hashValue {
		return hashSame
	}

	return hashDifferent
}

func (h BuildingBlockDefinitionVersionContentHash) toBase64() string {
	text := fmt.Sprintf("v%d:%s", h.hashVersion, h.hashValue)
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	return encoded
}

func (h *BuildingBlockDefinitionVersionContentHash) loadFromBase64(encoded string) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return errors.New("invalid hash format: expected 'version:value'")
	}

	hashVersion, err := strconv.Atoi(parts[0][1:]) // skip leading "v"
	if err != nil {
		return errors.New("invalid hash version: expected integer")
	}

	h.hashVersion = hashVersion
	h.hashValue = parts[1]

	return nil
}
