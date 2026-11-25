package provider

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/meshcloud/meshstack-cli/client"
)

type rawState map[string]jsontext.Value

type rawStateRewrite func(state rawState) error

// rawStateUpgrader rewrites the prior state as JSON, so a resource needs no copy of its old schema.
// It decodes only the objects along the paths it changes, which keeps every other value byte for byte.
func rawStateUpgrader(rewrites ...rawStateRewrite) resource.StateUpgrader {
	return resource.StateUpgrader{
		StateUpgrader: func(_ context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			upgraded, err := rewriteRawState(req.RawState.JSON, rewrites...)
			if err != nil {
				resp.Diagnostics.AddError("Unable to Upgrade Resource State", err.Error())
				return
			}
			resp.DynamicValue = &tfprotov6.DynamicValue{JSON: upgraded}
		},
	}
}

func rewriteRawState(prior []byte, rewrites ...rawStateRewrite) ([]byte, error) {
	var state rawState
	if err := json.Unmarshal(prior, &state); err != nil {
		return nil, err
	}
	for _, rewrite := range rewrites {
		if err := rewrite(state); err != nil {
			return nil, err
		}
	}
	return json.Marshal(state, json.Deterministic(true))
}

func moveStateAttribute(from, to []string) rawStateRewrite {
	return func(state rawState) error {
		value, found, err := state.get(from)
		if err != nil || !found {
			return err
		}
		if err := state.remove(from); err != nil {
			return err
		}
		return state.set(to, value)
	}
}

func removeStateAttribute(at ...string) rawStateRewrite {
	return func(state rawState) error {
		return state.remove(at)
	}
}

// projectRefFromState builds a project ref at `to` from the workspace and project name stored at the
// given paths, and removes both.
func projectRefFromState(workspace, name, to []string) rawStateRewrite {
	return func(state rawState) error {
		var ws, project *string
		for _, field := range []struct {
			path []string
			into **string
		}{{workspace, &ws}, {name, &project}} {
			value, found, err := state.get(field.path)
			if err != nil {
				return err
			}
			if found {
				if err := json.Unmarshal(value, field.into); err != nil {
					return err
				}
			}
			if err := state.remove(field.path); err != nil {
				return err
			}
		}
		if ws == nil || project == nil {
			return nil
		}
		ref, err := json.Marshal(map[string]string{
			"kind":               client.MeshObjectKind.Project,
			"owned_by_workspace": *ws,
			"name":               *project,
		}, json.Deterministic(true))
		if err != nil {
			return err
		}
		return state.set(to, ref)
	}
}

func (s rawState) get(path []string) (jsontext.Value, bool, error) {
	value, found := s[path[0]]
	if !found || len(path) == 1 {
		return value, found, nil
	}
	child, err := decodeRawState(value)
	if err != nil || child == nil {
		return nil, false, err
	}
	return child.get(path[1:])
}

func (s rawState) set(path []string, value jsontext.Value) error {
	if len(path) == 1 {
		s[path[0]] = value
		return nil
	}
	child, err := decodeRawState(s[path[0]])
	if err != nil {
		return err
	}
	if child == nil {
		child = rawState{}
	}
	if err := child.set(path[1:], value); err != nil {
		return err
	}
	return s.encode(path[0], child)
}

func (s rawState) remove(path []string) error {
	if len(path) == 1 {
		delete(s, path[0])
		return nil
	}
	child, err := decodeRawState(s[path[0]])
	if err != nil || child == nil {
		return err
	}
	if err := child.remove(path[1:]); err != nil {
		return err
	}
	return s.encode(path[0], child)
}

func (s rawState) encode(key string, child rawState) error {
	encoded, err := json.Marshal(child, json.Deterministic(true))
	if err != nil {
		return err
	}
	s[key] = encoded
	return nil
}

func decodeRawState(value jsontext.Value) (rawState, error) {
	if len(value) == 0 {
		return nil, nil
	}
	var state rawState
	err := json.Unmarshal(value, &state)
	return state, err
}
