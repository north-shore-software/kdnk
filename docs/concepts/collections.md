---
title: Collections
---

`kdnk.Collection[T](r, name, opts...)` declares a collection whose fields come from the
struct `T`, and returns a `*kdnk.Store[T]` for using it from Go.

```go
type Todo struct {
	Title   string  `json:"title" jsonschema:"What the task is"`
	Urgency float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
}

todos := kdnk.Collection[Todo](r, "todos")
```

**Fields.** Each exported field becomes a column, named by its `json` tag, or by the Go field name
if the tag has no name. Fields tagged `json:"-"` are skipped. The `jsonschema` tag is the field's
description in the generated tools. These types are supported, including named types with these
underlying kinds:

| Go type | PocketBase field |
|---|---|
| `string` | text |
| `bool` | bool |
| signed and unsigned integers | number, integers only |
| `float32`, `float64` | number |
| `time.Time` | date |

Embedded fields, any other type, the names `id`, `owner` and `created`, and two fields with the
same name make `Run` return an error.

**Hidden fields.** Every record also has an `owner`, the user who created it, and a `created`
time. Records are deleted when their owner is.

**Tables.** When the server starts serving, the table is created if it doesn't exist. If it does,
missing columns are added. A column whose type differs from the struct makes serving fail.
Columns are never changed or removed.

**Generated tools.** By default the model gets five tools for a collection called `todos`:

| Tool | Arguments | Result |
|---|---|---|
| `todos_create` | the struct's fields | the new record |
| `todos_list` | `limit`, optional, 1 to 100, default 20 | records, newest first |
| `todos_get` | `id` | the record |
| `todos_update` | `id` and all the struct's fields | the updated record |
| `todos_delete` | `id` | `{"deleted": "<id>"}` |

A record is returned to the model as the struct's JSON plus `id` and `created`. `todos_update`
replaces every field. Every tool only sees the session user's records; another user's ID is
reported as not found.

**Restrictions.** Pass an option to limit which tools are generated. The last one passed wins.
Restrictions apply to the model only; Go code can always use every method of the store.

| Option | Tools |
|---|---|
| none | create, list, get, update, delete |
| `kdnk.AppendOnly` | create, list, get |
| `kdnk.ReadOnly` | list, get |
| `kdnk.Hidden` | none |

**Using a collection from Go.** The `Store[T]` methods are scoped to a session's user, and work
once `Run` has started the app. Calling them earlier returns an error.

- `Create(ctx, s, v)` adds a record and returns it.
- `List(ctx, s, limit)` returns up to `limit` records, newest first. `limit` must be positive.
- `Get(ctx, s, id)` returns a record.
- `Update(ctx, s, id, v)` replaces a record's fields and returns it.
- `Delete(ctx, s, id)` removes a record.

`Get`, `Update` and `Delete` return `kdnk.ErrNotFound` if the record doesn't exist or
belongs to another user. Each `Record[T]` has `ID`, `Created` and `Data`.
