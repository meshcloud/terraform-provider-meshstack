package reflect

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
)

func Walk(v reflect.Value, visitor Visitor, opts ...WalkOption) error {
	return walk(walkOptions(opts).get().Root, v, visitor, opts)
}

type (
	// Visitor is provided to Walk as a callback function.
	// Descending further from this path on can be stopped
	// prematurely by returning the error from WalkPath.Stop.
	Visitor  func(path WalkPath, v reflect.Value) error
	WalkPath []*walkPathSegment
)

type (
	MapKey struct {
		reflect.Value
	}
	WithMapKeyVisitor = withVisitor[MapKey]
)

type (
	StructField struct {
		reflect.StructField
	}
	WithStructFieldVisitor = withVisitor[StructField]
)

type (
	WalkOption  func(*walkOpts)
	walkOptions []WalkOption
	walkOpts    struct {
		Root                    WalkPath
		VisitEmptyContainers    bool
		VisitEmbeddedNilStructs bool
	}
)

func WithRoot(root WalkPath) WalkOption {
	return func(w *walkOpts) {
		w.Root = root
	}
}

func VisitEmptyContainers() WalkOption {
	return func(w *walkOpts) {
		w.VisitEmptyContainers = true
	}
}

func VisitEmbeddedNilStructs() WalkOption {
	return func(w *walkOpts) {
		w.VisitEmbeddedNilStructs = true
	}
}

func (opts walkOptions) get() (result walkOpts) {
	for _, opt := range opts {
		opt(&result)
	}
	return
}

func (p WalkPath) String() (result string) {
	if p.IsRoot() {
		return "<root>"
	}
	for _, segment := range p {
		segment.joinString(&result)
	}
	return
}

func (p WalkPath) TryTraverse(other any) (traversed reflect.Value, err error) {
	// avoid double-wrapping for convenience, usually the last else case should hit!
	if otherValue, ok := other.(reflect.Value); ok {
		traversed = otherValue
	} else {
		traversed = reflect.ValueOf(other)
	}
	var errs []error
	for _, segment := range p {
		traversed, err = segment.tryDescend(traversed)
		errs = append(errs, err)
	}
	return traversed, errors.Join(errs...)
}

func (p WalkPath) Join(others ...WalkPath) (joined WalkPath) {
	joined = slices.Clone(p)
	for _, other := range others {
		joined = append(joined, other...)
	}
	return
}

var errStopWalking = errors.New("stop walking")

func (p WalkPath) Stop() error {
	return errStopWalking
}

func (p WalkPath) IsRoot() bool {
	return len(p) == 0
}

func walk(path WalkPath, v reflect.Value, visitor Visitor, opts []WalkOption) error {
	if err := visitor(path, v); errors.Is(err, errStopWalking) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.Join(
		path.WalkSlice(v, visitor, opts...),
		path.WalkMap(v, withVisitorOf[MapKey](visitor), opts...),
		path.WalkStruct(v, withVisitorOf[StructField](visitor), opts...),
		path.WalkPointer(v, visitor, opts...),
	)
}

func (p WalkPath) WalkSlice(v reflect.Value, visitor Visitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Array, reflect.Slice), v, func(canKinds []reflect.Kind) error {
		if walkOptions(opts).get().VisitEmptyContainers && v.Len() == 0 {
			return p.walk(v, visitor, opts, &walkPathSegment{
				Descend:    zeroElemValue,
				CanKinds:   canKinds,
				JoinString: joinStringWrapKeyOrIndex(pathZeroKeyOrIndex),
			})
		}
		for i := 0; i < v.Len(); i++ {
			segment := &walkPathSegment{
				Descend: func(other reflect.Value) reflect.Value {
					return other.Index(i)
				},
				CanKinds:   canKinds,
				JoinString: joinStringWrapKeyOrIndex(strconv.Itoa(i)),
			}
			if err := p.walk(v, visitor, opts, segment); err != nil {
				return err
			}
		}
		return nil
	})
}

type withVisitor[K any] func(WalkPath, *K, reflect.Value) error

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

func (k MapKey) Compare(other MapKey) int {
	return strings.Compare(k.Name(), other.Name())
}

func (k MapKey) Name() string {
	return fmt.Sprintf("%v", k.Interface())
}

func (p WalkPath) WalkMap(v reflect.Value, visitor WithMapKeyVisitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Map), v, func(canKinds []reflect.Kind) error {
		if walkOptions(opts).get().VisitEmptyContainers && v.Len() == 0 {
			return p.walk(v, toSingleShotVisitor(visitor, nil), opts, &walkPathSegment{
				Descend:    zeroElemValue,
				CanKinds:   canKinds,
				JoinString: joinStringWrapKeyOrIndex(pathZeroKeyOrIndex),
			})
		}

		for _, key := range iter.MapAndSortBy(newMapKey, iter.PickFirst(v.Seq2())) {
			segment := &walkPathSegment{
				Descend: func(other reflect.Value) reflect.Value {
					return other.MapIndex(key.Value)
				},
				CanKinds:   canKinds,
				JoinString: joinStringWrapKeyOrIndex(key.Name()),
			}
			if err := p.walk(v, toSingleShotVisitor(visitor, &key), opts, segment); err != nil {
				return err
			}
		}
		return nil
	})
}

func newStructField(field reflect.StructField) StructField {
	return StructField{field}
}

func (f StructField) Compare(other StructField) int {
	return strings.Compare(f.Name(), other.Name())
}

func (f StructField) Name() string {
	return f.StructField.Name
}

func (p WalkPath) WalkStruct(v reflect.Value, visitor WithStructFieldVisitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Struct), v, func(canKinds []reflect.Kind) error {
		visitEmbeddedNilStructs := walkOptions(opts).get().VisitEmbeddedNilStructs
		for _, field := range iter.MapAndSortBy(newStructField, slices.Values(reflect.VisibleFields(v.Type()))) {
			if !field.Anonymous && field.IsExported() {
				segment := &walkPathSegment{
					Descend: func(other reflect.Value) reflect.Value {
						return structFieldByIndex(visitEmbeddedNilStructs, other, field.Index)
					},
					CanKinds: canKinds,
					JoinString: func(s *string) {
						*s += pathSeparator + field.Name()
					},
				}
				if err := p.walk(v, toSingleShotVisitor(visitor, &field), opts, segment); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (p WalkPath) WalkPointer(v reflect.Value, visitor Visitor, opts ...WalkOption) error {
	return walkFor(kinds(reflect.Pointer, reflect.Interface), v, func(canKinds []reflect.Kind) error {
		if v.IsNil() {
			return walkFor(kinds(reflect.Pointer), v, func(_ []reflect.Kind) error {
				return p.walk(v, visitor, opts, &walkPathSegment{
					Descend:    zeroElemValue,
					CanKinds:   kinds(reflect.Pointer),
					JoinString: joinStringPointer,
				})
			})
		}

		return p.walk(v, visitor, opts, &walkPathSegment{
			Descend:    reflect.Value.Elem,
			CanKinds:   canKinds,
			JoinString: joinStringPointer,
		})
	})
}

func walkFor(canKinds []reflect.Kind, v reflect.Value, action func(canKinds []reflect.Kind) error) error {
	if slices.Contains(canKinds, v.Kind()) {
		return action(canKinds)
	}
	return nil
}

func (p WalkPath) walk(v reflect.Value, visitor Visitor, opts []WalkOption, segment *walkPathSegment) error {
	if descended := segment.Descend(v); descended.IsValid() {
		segment.current = append(slices.Clone(p), segment)
		return walk(segment.current, descended, visitor, opts)
	}
	return nil
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
						// invalid types are skipped in [WalkPath.walk] and lead to an error in tryDescend
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
	return reflect.Zero(v.Type().Elem())
}

const (
	pathSeparator      = "."
	pathIndexPrefix    = "["
	pathIndexSuffix    = "]"
	pathPointer        = '*'
	pathZeroKeyOrIndex = "<>"
)

func joinStringWrapKeyOrIndex(keyOrIndex string) func(*string) {
	return func(s *string) {
		*s += pathIndexPrefix + keyOrIndex + pathIndexSuffix
	}
}

func joinStringPointer(s *string) {
	insertPathPointerAt := func(pos int) {
		*s = string(slices.Insert([]byte(*s), pos, pathPointer))
	}
	idxPathSep := strings.LastIndex(*s, pathSeparator)
	idxIndexPrefix := strings.LastIndex(*s, pathIndexPrefix)
	if idxIndexPrefix > idxPathSep {
		insertPathPointerAt(idxIndexPrefix)
	} else {
		// Note: idxPathSep might be -1 (not found)
		// the pathPointer is then inserted into the beginning of s
		insertPathPointerAt(idxPathSep + 1)
	}
}

type walkPathSegment struct {
	Descend    func(other reflect.Value) reflect.Value
	CanKinds   []reflect.Kind
	JoinString func(s *string)

	current WalkPath
}

func (w walkPathSegment) tryDescend(other reflect.Value) (v reflect.Value, err error) {
	defer func() {
		// capture any panics to error as we're only trying to descend
		if panicked := recover(); panicked != nil {
			err = fmt.Errorf("%v", panicked)
		}
		// prepend with current path as context for error
		if err != nil {
			err = fmt.Errorf("path %s: %v", w.current, err)
		}
	}()

	otherKind := other.Kind()
	if slices.Contains(w.CanKinds, otherKind) {
		if otherDescend := w.Descend(other); otherDescend.IsValid() {
			return otherDescend, nil
		} else {
			return v, errors.New("encountered invalid value while descending")
		}
	} else {
		return v, fmt.Errorf("descend impossible on other value with kind %s", otherKind)
	}
}

func (w walkPathSegment) joinString(s *string) {
	w.JoinString(s)
}
