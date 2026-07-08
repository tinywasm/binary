# PLAN — binary: stop using codec `Uint` (migrate `Message.ID` to `Int`)

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

You are an external agent with **zero prior context** about this project. Everything you
need is in this file. Read it fully before writing code.

---

## Development Rules

- **No standard library for formatting/errors:** this module uses `github.com/tinywasm/fmt`
  (see existing imports). Do not introduce standard `fmt`/`strings`/`strconv`.
- **Do not run `gopush` or `codejob`** — local developer tools managed outside this task.
  You only modify code and tests, and run `go test ./...`.
- **Do not update `go.mod` dependencies** — in particular, do NOT bump
  `github.com/tinywasm/model`. This plan must compile against the currently pinned version.

---

## 1. Context and problem

The ecosystem decided that the codec numeric surface is **64-bit only**
(`int64` / `float64`) — see `model/docs/WHY_64BIT_ONLY.md` in the
`github.com/tinywasm/model` repo. As part of that decision, the `Uint` method is being
removed from the `model.FieldWriter` / `model.FieldReader` interfaces.

This module (`binary`) is the **only caller** of `Uint` in the whole ecosystem:
`Message` (the inter-module pub/sub envelope, `message.go`) encodes its correlation
`ID uint32` via `w.Uint("ID", uint64(m.ID))`.

That usage does not need `Uint`: a `uint32` (max ≈4.29×10⁹) fits losslessly in `int64`.
Migrating to `Int` unblocks the interface removal in `model`.

**Sequencing:** this plan runs FIRST, against the currently pinned `model` version (which
still has `Uint` in the interfaces). The removal in `model` happens afterwards, in a
separate plan in that repo.

**Wire-format note:** switching the `ID` field from the unsigned to the signed encoding
changes the bytes on the wire for that field. `Message` is an ephemeral pub/sub envelope —
never persisted — and both endpoints always ship from the same module version, so no
migration or compatibility shim is needed.

## 2. Changes

### 2.1 `message.go` — encode/decode `ID` via `Int`

```go
// EncodeFields — before:
w.Uint("ID", uint64(m.ID))
// after:
w.Int("ID", int64(m.ID))

// DecodeFields — before:
if id, ok := r.Uint("ID"); ok {
    m.ID = uint32(id)
}
// after:
if id, ok := r.Int("ID"); ok {
    m.ID = uint32(id)
}
```

`Message.ID` stays `uint32` — only the codec calls change.

### 2.2 Tests — remove every `Uint` call site

- `shared_test.go` (~lines 81, 95): the test fixture encodes/decodes an `ID` via
  `w.Uint` / `r.Uint`. Migrate to `Int` with the same cast pattern as 2.1 (adjust the
  fixture's field type or casts as needed — keep the fixture's intent).
- `codecs_test.go`: the `encodableUint64` / `decodableUint64` round-trip test
  (~lines 188-210) and the fixture at ~line 218 exercise the `Uint` codec path itself.
  **Delete the `Uint`-specific round-trip test types and their test cases** — the path is
  being removed from the ecosystem contract — and migrate any remaining fixture `Uint`
  calls to `Int`.
- After this plan, `grep -rn "\.Uint(" .` must return **zero** matches in this module
  (implementation methods `func (w *binaryWriter) Uint(...)` may remain — see 2.3).

### 2.3 Keep the `Uint` implementation methods — for now

`binaryWriter.Uint` (`codec.go` ~line 40) and `binaryReader.Uint` (~line 195) MUST stay:
the currently pinned `model` version still declares `Uint` in the interfaces, and removing
the methods would break interface satisfaction. Add this comment above each:

```go
// Deprecated: required only while model.FieldWriter/FieldReader still declare Uint.
// Delete when this module updates to the model version that removes it.
```

(adjust Writer/Reader naming per method).

## 3. Acceptance criteria

1. No `.Uint(` call sites remain in the module (prod or test code).
2. `Message.ID` round-trips losslessly through encode/decode as `uint32`.
3. `binaryWriter.Uint` / `binaryReader.Uint` still exist, marked deprecated.
4. `go.mod` untouched.
5. `go test ./...` passes.

## 4. Out of scope

- Deleting the `Uint` implementation methods (follow-up after `model` publishes the
  interface removal and this module bumps the dependency).
- Any change in the `model`, `json`, or `jsvalue` repos.

## 5. Stages

| # | Stage | Files | Output |
|---|-------|-------|--------|
| 1 | Migrate envelope | `message.go` | `ID` encoded/decoded via `Int` |
| 2 | Migrate/remove tests | `shared_test.go`, `codecs_test.go` | Zero `Uint` call sites; suite green |
| 3 | Deprecate impls | `codec.go` | Both `Uint` methods marked deprecated, kept |
