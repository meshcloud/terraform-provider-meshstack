package version

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
}

func Parse(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("cannot parse '%s' as version: expected 3, got %d fields separated by '.'", s, len(parts))
	}
	var errs []error
	partTo := func(i int, target *int) {
		parsed, err := strconv.Atoi(parts[i])
		if err == nil && parsed < 0 {
			err = fmt.Errorf("negative number '%d' not allowed", parsed)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("part i=%d: %w", i, err))
		} else {
			*target = parsed
		}
	}
	var result Version
	partTo(0, &result.Major)
	partTo(1, &result.Minor)
	partTo(2, &result.Patch)
	if len(errs) > 0 {
		return Version{}, fmt.Errorf("cannot parse '%s' as version: %w", s, errors.Join(errs...))
	}
	return result, nil
}

func MustParse(s string) Version {
	version, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return version
}

func (v Version) Compare(other Version) int {
	if major := cmp.Compare(v.Major, other.Major); major != 0 {
		return major
	} else if minor := cmp.Compare(v.Minor, other.Minor); minor != 0 {
		return minor
	}
	return cmp.Compare(v.Patch, other.Patch)
}

func (v Version) Less(other Version) bool {
	return v.Compare(other) < 0
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v *Version) UnmarshalJSON(bytes []byte) (err error) {
	var s string
	if err = json.Unmarshal(bytes, &s); err != nil {
		return
	}
	*v, err = Parse(s)
	return
}
