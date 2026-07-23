---
name: modern-go
description: Modern Go idioms used in this repo (Go 1.26) — the new(expression) builtin for inline pointers, the generics patterns the codebase relies on (typed clients, mock stores, variant unions, the generic TF value-conversion layer, map/iter helpers), and the go1.26 `go fix` modernizer pass. Use when writing or reviewing Go that creates pointers, defines type-parameterized helpers, touches generic.Set/Get, or running a modernization sweep.
---

# Modern Go in this repo

`go.mod` declares **`go 1.26`**. Two idioms matter most here: `new(expression)` for pointers, and
the codebase's generics. To keep the tree on those idioms, run the go1.26 [`go fix`](#go-fix--the-go126-modernizer-pass)
modernizer pass occasionally.

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

## `go fix` — the go1.26 modernizer pass

Go 1.26 reworked `go fix` into an analyzer-driven modernizer: each analyzer reports an
*opportunity for improvement* and carries a fix that is **safe to apply** (unlike `go vet`, which
only reports). It is how you keep the tree on the idioms above — e.g. it rewrites a stale
`interface{}` to `any` and a 3-clause counting loop to `for i := range`.

```bash
go tool fix help          # list registered analyzers (any, rangeint, omitzero, newexpr, minmax, …)
go fix -diff ./...        # preview every suggested fix as a unified diff — review before applying
go fix ./...              # apply them in place
```

Not part of CI or `task lint` (lint is golangci-lint only — see AGENTS.md "Lint policy"). Treat
`go fix` as an occasional, human-reviewed sweep, not an automated gate: some analyzers report
non-problems, so always read `-diff` first and commit the result as its own `chore`.

**Read each fix, don't rubber-stamp it.** The concrete pass on this repo (commit `chore: apply
go1.26 go fix idioms`) is the worked example:

- `any` — `interface{}` → `any` in a test helper signature. Pure syntax.
- `rangeint` — `for i := 0; i < len(tokens); i++` → `for i := range tokens`. Pure syntax.
- `omitzero` — the one needing judgement. It flagged `,omitempty` on struct-typed framework fields
  (`types.SecretOrAny`, `types.List`) and **stripped the tag** rather than take its own alternative
  fix (`,omitempty` → `,omitzero`, flagged "behavior change" and ignored by default). Correct here:
  `encoding/json` never omits *struct* types, so `,omitempty` on them was already dead — the field
  always serialized (`Variant` marshals its zero value to `null`), so dropping the tag is a no-op.
  This dovetails with the pointers rule above: `omitempty` earns its place only on genuinely
  nullable fields (pointers, slices, maps), never on a value-typed struct.

The `newexpr` analyzer (→ `new(expression)`) and `minmax` are also registered, so a future sweep
keeps the codebase aligned with the [`new(expression)`](#newexpression-for-pointers) idiom
automatically.
