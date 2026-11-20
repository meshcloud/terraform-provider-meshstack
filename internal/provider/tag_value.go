package provider

import (
	"maps"
	"slices"
	"strings"
)

func withTagValue(tags map[string][]string, key, value string) (updated map[string][]string, changed bool) {
	if slices.Contains(tags[key], value) {
		return tags, false
	}
	updated = maps.Clone(tags)
	if updated == nil {
		updated = map[string][]string{}
	}
	updated[key] = append(slices.Clone(tags[key]), value)
	return updated, true
}

func withoutTagValue(tags map[string][]string, key, value string) (updated map[string][]string, changed bool) {
	if !slices.Contains(tags[key], value) {
		return tags, false
	}
	updated = maps.Clone(tags)
	remaining := slices.DeleteFunc(slices.Clone(tags[key]), func(v string) bool { return v == value })
	if len(remaining) == 0 {
		delete(updated, key)
	} else {
		updated[key] = remaining
	}
	return updated, true
}

func parseTagValueImportID(id string, identifiers int) (ids []string, key, value string, ok bool) {
	parts := strings.SplitN(id, ".", identifiers+1)
	if len(parts) != identifiers+1 {
		return nil, "", "", false
	}
	key, value, found := strings.Cut(parts[identifiers], "=")
	if !found || key == "" || value == "" || slices.Contains(parts[:identifiers], "") {
		return nil, "", "", false
	}
	return parts[:identifiers], key, value, true
}
