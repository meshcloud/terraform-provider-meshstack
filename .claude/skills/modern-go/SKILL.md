---
name: modern-go
description: Modern Go idioms used in this repo (Go 1.26) — the new(expression) builtin for inline pointers, and the generics patterns the codebase relies on (typed clients, mock stores, variant unions, the generic TF value-conversion layer, map/iter helpers). Use when writing or reviewing Go that creates pointers, defines type-parameterized helpers, or touches generic.Set/Get.
---

# Modern Go in this repo

`go.mod` declares **`go 1.26`**. Two idioms matter most here: `new(expression)` for pointers, and
the codebase's generics.

## `new(expression)` for pointers

Go 1.26 extended the `new` builtin to accept an **expression**, not just a type — it allocates,
initializes, and returns a pointer in one step. Prefer it over helper functions (`ptr.To`,
`ptrTo` — these were removed and must not be reintroduced).

```go
s := new("hello")                          // *string
n := new(int64(1))                         // *int64
p := new(myStruct{A: "x"})                 // *myStruct
```

Real usage in this repo:

```go
secret.Hash    = new(fmt.Sprintf("sha256:%s", *secret.Plaintext)) // internal/clientmock/mock_client.go
dto.VersionNumber = new(int64(1))                                  // building_block_definition_resource_model.go
requestBody    = new(bytes.Buffer)                                 // client/internal/http_client.go
m[method]      = new(sync.Map)                                     // client/internal/retry.go
```

- Use it for inline pointer creation in struct literals, args, and returns.
- Works with any expression: `new(a + b)`, `new(convertSecret(in.Argument.X, "argument"))`.
- Chaining works: `new(new("v"))` → `**string`.
- Pair with the data-structure rule: pointers + `omitempty` only for fields **actually nullable**
  in the backend API; non-nullable fields use value types.

## Generics

The codebase leans on type parameters for type-safe domain and framework code. Reuse these rather
than writing `any`-typed or reflection-based variants.

### Domain / infra generics

| Type / func | File | Role |
|---|---|---|
| `MeshObjectClient[M any]`, `NewMeshObjectClient[M]`, `InferKind[M]()` | `client/internal/mesh_object_client.go` | Typed CRUD client per meshObject type |
| `Store[M any]` (`Get/Set/Delete/Values/SortedKeys`) | `internal/clientmock/mock_client.go` | Generic in-memory mock store; e.g. `NewStore[client.MeshBuildingBlockDefinitionVersion]()` |
| `Variant[X, Y any]` (custom `MarshalJSON`/`UnmarshalJSON`) | `client/types/variant/variant.go` | Discriminated union for JSON fields that are one-of-two |
| `Pollable[T any]`, `AtMostFor[T]`, `WithLastResultTo[T]` | `internal/util/poll/poll.go` | Timeout/retry polling abstraction |
| `NullIsUnknown[T any]`, `KnownValue[T]` | `internal/types/generic/unknown.go` | Terraform null-vs-unknown handling |

### The `generic` TF value-conversion layer

`internal/types/generic/` converts between Go structs and Terraform values generically — this is
how resources read plan/config and write state:

```go
generic.Set[T](ctx, setter, in, opts...)   diag.Diagnostics   // set.go      — write state
generic.Get[T](ctx, getter, diags, opts...) (out T)           // get.go      — read plan/config
generic.ValueTo[T](in, opts...)   (T, error)                  // value_to.go — tftypes.Value → T
generic.ValueFrom[T](in, opts...) (tftypes.Value, error)      // value_from.go
```

Customize conversion for a specific type with `WithValueToConverterFor[T]` /
`WithValueFromConverterFor[T]` (see `building_block_definition_resource_model.go` for
`SecretOrAny` handling). Default to `generic.Set`/`generic.Get` in resources; reach for the
converter options only when a field needs bespoke (de)serialization.

### Utility generics

```go
maps.SortedFunc[K comparable, V any](m, cmp)      // internal/util/maps/maps.go — iter.Seq2 in sorted order
maps.MapValues[K comparable, From, To any](m, f)  // map a map's values
iter.PickFirst / iter.Map / iter.MapAndSortBy     // internal/util/iter/iter.go
```

### Constraint patterns in use

- `[T any]` — the common case.
- `[K comparable, V any]` — map keys.
- `[X, Y any]` — multiple independent params (`Variant`).
- `[T any, Comparable interface{ Compare(Comparable) int }]` — method-bearing constraint
  (`MapAndSortBy`).

Prefer `comparable` / a small method interface over `any` when the function actually requires it —
it pushes misuse to compile time.
