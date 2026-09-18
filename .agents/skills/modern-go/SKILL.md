---
name: modern-go
description: Modern Go idioms used in this repo (Go 1.27) — the new(expression) builtin for inline pointers, the encoding/json/v2 rules (pointer + `,omitzero` for a nullable field, the nil slice/map option bundle), the generics patterns the codebase relies on (typed clients, mock stores, variant unions, the generic TF value-conversion layer, map/iter helpers), and the `go fix` modernizer pass. Use when writing or reviewing Go that creates pointers, adds or changes a json struct tag, marshals a request body or a state attribute, defines type-parameterized helpers, touches generic.Set/Get, or running a modernization sweep.
---

# Modern Go in this repo

`go.mod` declares **`go 1.27`**. Two idioms matter most here: `new(expression)` for pointers, and
the codebase's generics. To keep the tree on those idioms, run the [`go fix`](#go-fix--the-modernizer-pass)
modernizer pass occasionally.

Bumping the Go version is a **two-file** change: `flake.nix` pins the toolchain (`go_1_27` and the
`GOROOT` derived from it), so a `go.mod` bump must update the flake's pin in lock-step — otherwise
`nix develop` builds against a different Go than `go.mod` targets. Add `nix flake update` when the
locked nixpkgs is too old to carry the new attribute, which fails as
`undefined variable 'go_1_NN'`.

The bump also changes what `task lint` enforces, because golangci-lint is a `tool` directive in
`go.mod` and is therefore rebuilt with the new Go: its formatters use the `go/format` compiled into
the binary, so a new Go release can reformat code the old one accepted. Run `task lint -- --fix`
right after the bump and commit the result with it.

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
```

- Use it for inline pointer creation in struct literals, args, and returns.
- Works with any expression: `new(a + b)`, `new(convertSecret(in.Argument.X, "argument"))`.
- Chaining works: `new(new("v"))` → `**string`.

## JSON: `encoding/json/v2`

Everything here encodes and decodes with `encoding/json/v2`; depguard denies `encoding/json` in
every file. meshstack-cli's `client` package is the reference for both rules below — it carries the
DTOs and the marshalling this provider sends.

**Nullability: pointer + `,omitzero`, only when actually nullable.** A field that is genuinely
nullable in the backend API takes a pointer and `,omitzero`; any other field takes a value type and
no tag. Never pair a pointer with `,omitempty`: v2 omits whatever *encodes* as `null`, `""`, `{}` or
`[]`, so a `*string` pointing at `""` is dropped — and the empty string is how a caller clears an
optional field. For the same reason a value-typed struct whose marshaler emits `null`, such as a
`Variant`, takes no tag either.

**Nil slices and maps need the option bundle.** v2 writes a nil slice as `[]` and a nil map as `{}`,
and the backend reads that differently from `null` — `supportedPlatforms` on a workspace-level
building block definition rejects `[]` with a 400. So every marshal whose bytes reach the wire or
Terraform state passes `wireCompatibility` (`internal/provider/json.go`), and the client holds
itself to the same shape. Options do not travel with a value, so an intermediate marshal inside a
round-trip needs the bundle too, and a type that defines its own `MarshalJSON` repeats it.

## Generics

The codebase leans on type parameters for type-safe domain and framework code. Reuse these rather
than writing `any`-typed or reflection-based variants.

### Domain / infra generics

| Type / func | File | Role |
|---|---|---|
| `MeshObjectClient[M any]`, `NewMeshObjectClient[M]`, `InferKind[M]()` | meshstack-cli `client/internal/mesh_object_client.go` | Typed CRUD client per meshObject type |
| `Store[M any]` (`Get/Set/Delete/Values/SortedKeys`) | `internal/clientmock/mock_client.go` | Generic in-memory mock store; e.g. `NewStore[client.MeshBuildingBlockDefinitionVersion]()` |
| `Variant[X, Y any]` (custom `MarshalJSON`/`UnmarshalJSON`) | meshstack-cli `client/types/variant/variant.go` | Discriminated union for JSON fields that are one-of-two |
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

## `go fix` — the modernizer pass

Go 1.26 reworked `go fix` into an analyzer-driven modernizer: each analyzer reports an
*opportunity for improvement* and carries a fix that is **safe to apply** (unlike `go vet`, which
only reports). It is how you keep the tree on the idioms above — e.g. it rewrites a stale
`interface{}` to `any` and a 3-clause counting loop to `for i := range`.

```bash
go tool fix help          # list registered analyzers (any, rangeint, omitzero, newexpr, minmax, …)
go fix -diff ./...        # preview every suggested fix as a unified diff — review before applying
go fix ./...              # apply them in place
```

Not part of CI or `task lint` (lint is golangci-lint only — see AGENTS.md "Always-on rules"). Treat
`go fix` as an occasional, human-reviewed sweep, not an automated gate: some analyzers report
non-problems, so always read `-diff` first and commit the result as its own `chore`.

**Read each fix, don't rubber-stamp it.** The concrete pass on this repo (commit `chore: apply
go1.26 go fix idioms`) is the worked example:

- `any` — `interface{}` → `any` in a test helper signature. Pure syntax.
- `rangeint` — `for i := 0; i < len(tokens); i++` → `for i := range tokens`. Pure syntax.
- `omitzero` — the one needing judgement. It flagged `,omitempty` on struct-typed framework fields
  (`types.SecretOrAny`, `types.List`) and **stripped the tag** rather than take its own alternative
  fix (`,omitempty` → `,omitzero`, flagged "behavior change" and ignored by default). Stripping is
  what the [JSON rules](#json-encodingjsonv2) ask for, but decide each field against them rather
  than taking the analyzer's word: a "behavior change" note here is a question, not a verdict.

The `newexpr` analyzer (→ `new(expression)`) and `minmax` are also registered, so a future sweep
keeps the codebase aligned with the [`new(expression)`](#newexpression-for-pointers) idiom
automatically.
