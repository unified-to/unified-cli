# Object catalog

## Why

The CLI used to treat the object argument as opaque: `SplitObject` cut
`ats_candidate` at the first underscore and the two halves went straight into
the URL. Anything shaped like `x_y` was accepted, so a typo, an object that does
not exist, or a method the object does not support all became an HTTP error from
the API — or, for list filters, no error at all, because the API ignores query
parameters it does not recognise rather than rejecting them.

The CLI now carries a catalog of the objects the API exposes, what each one
supports, and which list filters it accepts.

## Where the data comes from

`internal/catalog/objects_gen.go` is generated from a checkout of
[unified-api](https://github.com/unified-to/unified-api) by
`internal/catalog/gen`:

```sh
go generate ./internal/catalog/...          # expects ../unified-api
go run ./internal/catalog/gen -api /path/to/unified-api -out internal/catalog/objects_gen.go
```

The API registers object routes in two ways, and the generator reads both:

- **`setRoute()`** in `src/server/routes/<category>/index.ts` covers all but a
  handful of objects. Each call names the object, the CRUD letters it exposes
  (`L`/`R`/`C`/`U`/`D`), its `list_params`, and any `list_params_enums` — the
  last of these often a spread of a `Unified*.ts` const, which the generator
  resolves to its values.
- **Hand-written `addRoute()` calls** cover `enrich_person`, `enrich_company`,
  `assessment_package`, `scim_users` and `scim_groups`. These have no single
  shape to parse, so their entries are written out in the generator's
  `specialRoutes` table. Each entry names the source file and the route paths it
  expects to find there; the generator checks for those paths and fails if they
  have moved, so the table cannot drift unnoticed.

`passthrough` is in the API's `ObjectType` list but is not addressable as
`/{category}/{connection_id}/{resource}`, so it has no catalog entry. The
`unified passthrough` command covers it instead.

Finally the generator reconciles what it found against `ObjectType` in
`src/models/Unified.ts` and fails if either side has an object the other does
not. An object added to the API therefore shows up as a generator failure rather
than as a silent gap.

TypeScript comments are stripped before parsing: several route files keep
commented-out `setRoute()` calls for objects that are not live yet.

## What the CLI does with it

- `unified objects [category|object]` browses the catalog, with `--json` for
  machine use.
- `list`, `get`, `create`, `update` and `remove` resolve the object through the
  catalog, so an unknown object gets a "did you mean" hint and an unsupported
  method is named along with the ones that do work.
- `list` also checks its query parameters against the object's filters.
- Shell completion offers the object names that support the command being run.

## Escape hatch

`--no-validate` skips every one of those checks: the object name is split as
before and the request goes out as written. It is there so that an object or
filter the API has released, but this CLI has not yet regenerated, is still
reachable. Every validation error names it.
