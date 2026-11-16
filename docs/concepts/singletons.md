---
title: Singletons
---

A singleton is a single value whose fields come from the struct `T`. There are two kinds:

- `kdnk.Singleton[T](r, name, opts...)` declares a per-user singleton, one value per user, and
  returns a `*kdnk.Value[T]`. The model can read and set it.
- `kdnk.Global[T](r, name, opts...)` declares a global singleton, one value shared by every user,
  and returns a `*kdnk.GlobalValue[T]`. The model can read it; only Go code can set it.

```go
type Prefs struct {
	Tone string `json:"tone" jsonschema:"How the user wants replies written"`
}

type Notice struct {
	Text string `json:"text" jsonschema:"An announcement for all users"`
}

prefs := kdnk.Singleton[Prefs](r, "prefs")
notice := kdnk.Global[Notice](r, "notice")
```

**Fields.** `T` follows the same rules as a [collection](/concepts/collections/)'s struct. The
names `id`, `owner`, `created` and `version` are reserved.

**Unset values.** A value that was never set reads as the zero `T`. A field added to `T` later
reads as its zero value on values stored before it existed.

**Tables.** Each singleton is a table named `name`, created and extended as for a collection. A
per-user singleton stores one row per user, with a hidden `owner`, and the row is deleted when its
user is. A global singleton stores one row with the ID `global000000000`. A singleton's name must
not clash with a collection or another singleton.

**Generated tools.** For a per-user singleton called `prefs` the model gets:

| Tool | Arguments | Result |
|---|---|---|
| `prefs_get` | none | the session user's value |
| `prefs_set` | `version` and all the struct's fields | the new value |

A global singleton gets only `notice_get`. A value is returned to the model as the struct's JSON
plus `version`, which is derived from the value's fields.

`prefs_set` replaces every field. It takes the `version` the model last read. If the value has
changed since, it changes nothing and returns the current value:

```json
{"error": "prefs was modified since it was read", "current": {"tone": "formal", "version": "…"}}
```

**Restrictions.** Pass an option to limit which tools are generated. The last one passed wins.
Restrictions apply to the model only.

| Option | Per-user tools | Global tools |
|---|---|---|
| none | get, set | get |
| `kdnk.SingletonReadOnly` | get | get |
| `kdnk.SingletonHidden` | none | none |

**Using a singleton from Go.** The methods work once `Run` has started the app; calling them
earlier returns an error. Go writes are not checked against a version.

- `Value[T].Get(ctx, s)` returns the session user's value.
- `Value[T].Set(ctx, s, v)` replaces the session user's value.
- `GlobalValue[T].Get(ctx)` returns the value.
- `GlobalValue[T].Set(ctx, v)` replaces the value.
