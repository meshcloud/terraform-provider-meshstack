package testconfig

import (
	"strings"
)

// Traversal is a sequence of HCL attribute path segments forming a resource address,
// e.g. Traversal{"meshstack_workspace", "my_ws"} → "meshstack_workspace.my_ws".
// It mirrors hcl.Traversal at a string level — just enough for test reference construction.
type Traversal []string

// String joins segments with ".".
func (t Traversal) String() string {
	return strings.Join(t, ".")
}

// Join appends additional segments, returning a new Traversal.
func (t Traversal) Join(elems ...string) Traversal {
	result := make(Traversal, len(t)+len(elems))
	copy(result, t)
	copy(result[len(t):], elems)
	return result
}
