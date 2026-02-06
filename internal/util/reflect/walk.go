package reflect

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
)

// Walk traverses a [reflect.Value] recursively, calling the visitor for each value encountered.
func Walk(v reflect.Value, visitor Visitor, opts ...WalkOption) error {
	return walk(walkOptions(opts).get().Root, v, visitor, opts)
}

// Visitor is provided to Walk as a callback function.
// Descending further from this path on can be stopped
// prematurely by returning the error from WalkPath.Stop.
type Visitor func(path WalkPath, v reflect.Value) error

// WalkOption configures Walk behavior.
type WalkOption func(*walkOpts)

// WithRoot sets the initial path for walking.
func WithRoot(root WalkPath) WalkOption {
	return func(w *walkOpts) {
		w.Root = root
	}
}

// VisitEmptyContainers enables visiting empty slices, arrays, and maps with a synthetic zero-element step.
func VisitEmptyContainers() WalkOption {
	return func(w *walkOpts) {
		w.VisitEmptyContainers = true
	}
}

// VisitEmbeddedNilStructs enables visiting fields of nil embedded struct pointers using zero values.
func VisitEmbeddedNilStructs() WalkOption {
	return func(w *walkOpts) {
		w.VisitEmbeddedNilStructs = true
	}
}

// WalkPath represents a path through a nested data structure.
type WalkPath []WalkPathStep

// String returns a human-readable representation of the path.
func (p WalkPath) String() (result string) {
	if p.IsRoot() {
		return "<root>"
	}
	const (
		pathSeparator      = "."
		pathIndexPrefix    = "["
		pathIndexSuffix    = "]"
		pathPointer        = '*'
		pathZeroKeyOrIndex = "<>"
	)

	joinStringWrapKeyOrIndex := func(keyOrIndex any) {
		result += fmt.Sprintf("%s%v%s", pathIndexPrefix, keyOrIndex, pathIndexSuffix)
	}

	joinStringPointer := func() {
		insertPathPointerAt := func(pos int) {
			result = string(slices.Insert([]byte(result), pos, pathPointer))
		}
		idxPathSep := strings.LastIndex(result, pathSeparator)
		idxIndexPrefix := strings.LastIndex(result, pathIndexPrefix)
		if idxIndexPrefix > idxPathSep {
			insertPathPointerAt(idxIndexPrefix)
		} else {
			// Note: idxPathSep might be -1 (not found)
			// the pathPointer is then inserted into the beginning of s
			insertPathPointerAt(idxPathSep + 1)
		}
	}

	for _, step := range p {
		switch s := step.(type) {
		case DerefPointer:
			joinStringPointer()
		case MapKey:
			if s.IsValid() {
				joinStringWrapKeyOrIndex(s.Name())
			} else {
				joinStringWrapKeyOrIndex(pathZeroKeyOrIndex)
			}
		case StructField:
			result += pathSeparator + s.Name
		case SliceIndex:
			if s < 0 {
				joinStringWrapKeyOrIndex(pathZeroKeyOrIndex)
			} else {
				joinStringWrapKeyOrIndex(s)
			}
		}
	}
	return
}

// TryTraverse attempts to apply this path to another value, returning the traversed value or an error.
func (p WalkPath) TryTraverse(other any) (traversed reflect.Value, err error) {
	var path WalkPath
	defer func() {
		// capture any panics to error as we're only trying to descend
		if panicked := recover(); panicked != nil {
			err = fmt.Errorf("%v", panicked)
		}
		// prepend with current path as context for error
		if err != nil {
			err = fmt.Errorf("path %s: %v", path, err)
		}
	}()
	// avoid double-wrapping for convenience, usually the last else case should hit!
	if otherValue, ok := other.(reflect.Value); ok {
		traversed = otherValue
	} else {
		traversed = reflect.ValueOf(other)
	}
	for _, step := range p {
		path = append(path, step)
		traversed = step.StepInto(traversed)
	}
	return
}

// Join concatenates this path with other paths, returning a new combined path.
func (p WalkPath) Join(others ...WalkPath) (joined WalkPath) {
	joined = slices.Clone(p)
	for _, other := range others {
		joined = append(joined, other...)
	}
	return
}

// Stop returns an error that signals the walk to stop descending further.
func (p WalkPath) Stop() error {
	return errStopWalking
}

// IsRoot returns true if this path is empty (represents the root).
func (p WalkPath) IsRoot() bool {
	return len(p) == 0
}

// WalkSlice walks through array and slice elements using SliceIndex.StepInto.
func (p WalkPath) WalkSlice(v reflect.Value, visitor WithSliceIndexVisitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Array, reflect.Slice), v, func(canKinds []reflect.Kind) error {
		if walkOptions(opts).get().VisitEmptyContainers && v.Len() == 0 {
			empty := SliceIndex(-1)
			return p.walk(v, toSingleShotVisitor(visitor, &empty), opts, empty)
		}
		for i := SliceIndex(0); i < SliceIndex(v.Len()); i++ {
			if err := p.walk(v, toSingleShotVisitor(visitor, &i), opts, i); err != nil {
				return err
			}
		}
		return nil
	})
}

// WalkMap walks through map key-value pairs, providing the key to the visitor using MapKey.StepInto.
func (p WalkPath) WalkMap(v reflect.Value, visitor WithMapKeyVisitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Map), v, func(canKinds []reflect.Kind) error {
		if walkOptions(opts).get().VisitEmptyContainers && v.Len() == 0 {
			empty := MapKey{reflect.Value{}}
			return p.walk(v, toSingleShotVisitor(visitor, &empty), opts, empty)
		}
		for _, key := range iter.MapAndSortBy(newMapKey, iter.PickFirst(v.Seq2())) {
			if err := p.walk(v, toSingleShotVisitor(visitor, &key), opts, key); err != nil {
				return err
			}
		}
		return nil
	})
}

// WalkStruct walks through exported struct fields, providing field metadata to the visitor using StructField.StepInto.
func (p WalkPath) WalkStruct(v reflect.Value, visitor WithStructFieldVisitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Struct), v, func(canKinds []reflect.Kind) error {
		visitEmbeddedNilStructs := walkOptions(opts).get().VisitEmbeddedNilStructs
		newStructField := func(field reflect.StructField) StructField {
			return StructField{field, visitEmbeddedNilStructs}
		}
		for _, field := range iter.MapAndSortBy(newStructField, slices.Values(reflect.VisibleFields(v.Type()))) {
			if !field.Anonymous && field.IsExported() {
				if err := p.walk(v, toSingleShotVisitor(visitor, &field), opts, field); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// WalkPointer walks through pointer and interface values, dereferencing them if non-nil,
// otherwise falling back to the zero value pointed to, see DerefPointer.StepInto.
func (p WalkPath) WalkPointer(v reflect.Value, visitor Visitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Pointer, reflect.Interface), v, func(canKinds []reflect.Kind) error {
		if v.IsNil() {
			return p.walk(v, visitor, opts, DerefPointer(nil))
		}
		return p.walk(v, visitor, opts, DerefPointer(reflect.Value.Elem))
	})
}

func (p WalkPath) WrapError(err error) error {
	if err == nil {
		return nil
	}
	var walkErr walkPathError
	if errors.As(err, &walkErr) {
		return err
	}
	return walkPathError{WalkPath: p, Wrapped: err}
}

// WalkPathStep represents a single step in a WalkPath and knows how to descend into values.
type WalkPathStep interface {
	// StepInto applies this step to a value, returning the descended value.
	StepInto(other reflect.Value) reflect.Value
}

// MapKey represents a map key step in a WalkPath.
type MapKey struct {
	reflect.Value
}

// WithMapKeyVisitor is a visitor that receives map key metadata.
type WithMapKeyVisitor = withVisitor[MapKey]

// StepInto retrieves the map value at this key, or returns a zero element for invalid keys.
func (k MapKey) StepInto(other reflect.Value) reflect.Value {
	if k.EmptyContainer() {
		return zeroElemValue(other)
	}
	return other.MapIndex(k.Value)
}

// Compare returns the lexicographic comparison of this key's name with another.
func (k MapKey) Compare(other MapKey) int {
	return strings.Compare(k.Name(), other.Name())
}

// Name returns the string representation of this map key (similar to StructField.Name).
func (k MapKey) Name() string {
	return fmt.Sprintf("%v", k.Interface())
}

// EmptyContainer tells if WalkPath.WalkMap is running in an empty container with VisitEmptyContainers.
func (k MapKey) EmptyContainer() bool {
	return !k.IsValid()
}

// StructField represents a struct field step in a WalkPath.
type StructField struct {
	reflect.StructField
	visitEmbeddedNilStructs bool
}

// WithStructFieldVisitor is a visitor that receives struct field metadata.
type WithStructFieldVisitor = withVisitor[StructField]

// StepInto retrieves the struct field value at this field's index.
func (f StructField) StepInto(other reflect.Value) reflect.Value {
	return structFieldByIndex(f.visitEmbeddedNilStructs, other, f.Index)
}

// Compare returns the lexicographic comparison of this field's name with another.
func (f StructField) Compare(other StructField) int {
	return strings.Compare(f.Name, other.Name)
}

// SliceIndex represents a slice/array index step in a WalkPath. Negative values represent zero elements.
type SliceIndex int

// WithSliceIndexVisitor is a visitor that receives slice index metadata.
type WithSliceIndexVisitor = withVisitor[SliceIndex]

// StepInto retrieves the element at this index, or returns a zero element for negative indices.
func (s SliceIndex) StepInto(other reflect.Value) reflect.Value {
	if s.EmptyContainer() {
		return zeroElemValue(other)
	}
	return other.Index(int(s))
}

// EmptyContainer tells if WalkPath.WalkSlice is running in an empty container with VisitEmptyContainers.
func (s SliceIndex) EmptyContainer() bool {
	return s < 0
}

// DerefPointer represents a pointer dereference step in a WalkPath, implemented as a transformation function.
type DerefPointer func(reflect.Value) reflect.Value

// StepInto applies the pointer dereference transformation to the value.
func (p DerefPointer) StepInto(other reflect.Value) reflect.Value {
	if p.IsNil() {
		return zeroElemValue(other)
	}
	return p(other)
}

func (p DerefPointer) IsNil() bool {
	return p == nil
}

// Private implementation details

var errStopWalking = errors.New("stop walking")

type (
	walkOptions []WalkOption
	walkOpts    struct {
		Root                    WalkPath
		VisitEmptyContainers    bool
		VisitEmbeddedNilStructs bool
	}
	withVisitor[K any] func(WalkPath, *K, reflect.Value) error
)

func (opts walkOptions) get() (result walkOpts) {
	for _, opt := range opts {
		opt(&result)
	}
	return
}

func (p WalkPath) walk(v reflect.Value, visitor Visitor, opts []WalkOption, step WalkPathStep) error {
	if descended := step.StepInto(v); descended.IsValid() {
		return walk(append(slices.Clone(p), step), descended, visitor, opts)
	}
	return nil
}

func walk(path WalkPath, v reflect.Value, visitor Visitor, opts []WalkOption) error {
	if err := visitor(path, v); errors.Is(err, errStopWalking) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.Join(
		path.WalkSlice(v, withVisitorOf[SliceIndex](visitor), opts...),
		path.WalkMap(v, withVisitorOf[MapKey](visitor), opts...),
		path.WalkStruct(v, withVisitorOf[StructField](visitor), opts...),
		path.WalkPointer(v, visitor, opts...),
	)
}

func walkFor(canKinds []reflect.Kind, v reflect.Value, action func(canKinds []reflect.Kind) error) error {
	if slices.Contains(canKinds, v.Kind()) {
		return action(canKinds)
	}
	return nil
}

func withVisitorOf[K any, V withVisitor[K]](visitor Visitor) V {
	return V(func(path WalkPath, k *K, v reflect.Value) error {
		return visitor(path, v)
	})
}

func toSingleShotVisitor[K any](visitor withVisitor[K], k *K) Visitor {
	used := false
	return func(path WalkPath, v reflect.Value) error {
		if used {
			return visitor(path, nil, v)
		} else {
			used = true
			return visitor(path, k, v)
		}
	}
}

func newMapKey(value reflect.Value) MapKey {
	return MapKey{value}
}

func kinds(ks ...reflect.Kind) []reflect.Kind {
	return ks
}

// structFieldByIndex is inspired by [reflect.Value.FieldByIndexErr].
func structFieldByIndex(fallbackToZero bool, v reflect.Value, index []int) reflect.Value {
	if len(index) == 1 {
		return v.Field(index[0])
	}
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					if fallbackToZero {
						// This is the fallback: get zero value of pointed-to-type instead of throwing error/panic.
						// If the Visitor plans to update field values, this will eventually panic then,
						// but investigating the value's structure is possible!
						v = zeroElemValue(v)
					} else {
						// invalid types are skipped in [WalkPath.walk] and lead to an error in [WalkPath.TryTraverse]
						return reflect.Value{}
					}
				} else {
					v = v.Elem()
				}
			}
		}
		v = v.Field(x)
	}
	return v
}

func zeroElemValue(v reflect.Value) reflect.Value {
	typ := v.Type()
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return reflect.Zero(typ.Elem())
	default:
		// For types without an element type (e.g., interface{}), return an invalid value
		// This will stop further descent in walk()
		return reflect.Value{}
	}
}

type walkPathError struct {
	WalkPath
	Wrapped error
}

func (e walkPathError) Error() (result string) {
	result = "path " + e.String()
	if e.Wrapped != nil {
		result += ": " + e.Wrapped.Error()
	}
	return
}

func (e walkPathError) Unwrap() error {
	return e.Wrapped
}
