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

func splitTagImportID(id string, identifiers int, withRest bool) (ids []string, rest string, ok bool) {
	parts := strings.SplitN(id, ".", identifiers+1)
	want := identifiers
	if withRest {
		want++
	}
	if len(parts) != want || slices.Contains(parts, "") {
		return nil, "", false
	}
	if withRest {
		rest = parts[identifiers]
	}
	return parts[:identifiers], rest, true
}

func parseTagValueImportID(id string, identifiers int) (ids []string, key, value string, ok bool) {
	ids, rest, ok := splitTagImportID(id, identifiers, true)
	key, value, found := strings.Cut(rest, "=")
	if !ok || !found || key == "" || value == "" {
		return nil, "", "", false
	}
	return ids, key, value, true
}

func tagValueNote(object string) string {
	return "~> **Note:** Use one `meshstack_" + object + "_tag_value` resource per value. The resource adds its value to the " +
		"tag and removes only that value again, so several of them can share a tag — but two resources declaring the same " +
		"value share it, too: destroying either removes it, and the other drops out of state and adds it again on the next " +
		"apply. A value that is already on the tag when this resource is created is adopted, and destroying the resource " +
		"removes it. Use this resource only for a tag whose definition is multi-select: recent meshStack versions reject a " +
		"second value on a tag that is not multi-select, and older versions store it. Do not combine this resource with " +
		"`meshstack_" + object + "_tag` for the same key, or with `meshstack_" + object + "_tags` or inline `tags` on " +
		"`meshstack_" + object + "` for the same " + object + "."
}
