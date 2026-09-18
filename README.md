# unified

A CLI tool for accessing the [Unified.to](https://unified.to) API. Perform CRUD operations on any Unified.to object from your terminal.

## Install

**Homebrew** (macOS):

```sh
brew tap unified-to/cli
brew install unified
```

**Go install**:

```sh
go install github.com/unified-to/unified-cli@latest
```

## Configuration

Provide your API key via the `--api-key` flag or the `UNIFIED_API_KEY` environment variable:

```sh
export UNIFIED_API_KEY="your-api-key"
```

Or pass it per command:

```sh
unified list abc123 ats_candidate --api-key "your-api-key"
```

## Usage

```
unified <method> <connection_id> <object> [flags]
```

Methods: `list`, `get`, `create`, `update`, `remove`

Not every object supports every method, and each object accepts its own list
filters. `unified objects` knows both, and the CRUD commands check the object
and its list filters before sending anything.

### List objects

```sh
unified list abc123 ats_candidate --limit 10
```

An object's own list filters are passed as flags too:

```sh
unified list abc123 ats_document --limit 10 --job_id J123 --type RESUME
```

### Get a single object

```sh
unified get abc123 ats_candidate --id ID456
```

### Create an object

Inline JSON:

```sh
unified create abc123 crm_contact -d '{"name":"John"}'
```

From a file:

```sh
unified create abc123 crm_contact -d @contact.json
```

### Update an object

```sh
unified update abc123 crm_contact --id ID456 -d '{"name":"Jane"}'
```

### Remove an object

```sh
unified remove abc123 crm_contact --id ID456
```

### Call a provider API directly

`passthrough` sends a request straight to the third-party API behind a
connection, with Unified.to handling authentication:

```sh
unified passthrough GET abc123 v1/contacts
unified passthrough GET abc123 "v1/contacts?updated_since=2026-01-01"
unified passthrough POST abc123 v1/contacts -d '{"name":"John"}'
```

## Discovering objects

`unified objects` lists every object the API supports, grouped by category:

```sh
unified objects              # all objects and the methods each supports
unified objects ats          # just the ats category
unified objects ats_document # endpoints and list filters for one object
unified objects --json       # the same data, machine-readable
```

```
$ unified objects ats_document
ats_document

  list    GET     /ats/{connection_id}/document
  get     GET     /ats/{connection_id}/document/{id}
  create  POST    /ats/{connection_id}/document
  update  PATCH   /ats/{connection_id}/document/{id}
  remove  DELETE  /ats/{connection_id}/document/{id}

List filters:
  --application_id
  --candidate_id
  --job_id
  --type  one of: RESUME, COVER_LETTER, OFFER_PACKET, OFFER_LETTER, TAKE_HOME_TEST, OTHER

Standard list parameters: --query --limit --offset --updated-gte --sort --order --fields --raw
```

## Object naming

Object names use the format `category_objectname`, which maps to the API path
`/{category}/{connection_id}/{object}`:

- `ats_candidate` -> category `ats`, object `candidate`
- `crm_contact` -> category `crm`, object `contact`
- `accounting_creditmemo` -> category `accounting`, object `creditmemo`

## Validation

Before sending a request the CLI checks that the object exists, that it supports
the method you asked for, and that the list filters you passed are ones the
object accepts. The API silently ignores query parameters it does not recognise,
so a mistyped filter would otherwise come back as an unfiltered result:

```
$ unified list abc123 ats_candidate --job_i J123
Error: ats_candidate does not accept the list parameter "job_i"

Accepted: company_id, fields, limit, offset, order, query, raw, sort, updated_gte

Run "unified objects ats_candidate" for details, or pass --no-validate to send the request anyway.
```

Pass `--no-validate` to skip these checks, which is what you want if the API has
released an object or filter that this CLI does not know about yet.

## Shell completion

Cobra generates completions for your shell, including object names:

```sh
unified completion zsh > "${fpath[1]}/_unified"   # zsh
unified completion bash > /etc/bash_completion.d/unified
```

## Output

JSON is printed to stdout. Errors go to stderr. Pipe to `jq` for formatting:

```sh
unified list abc123 ats_candidate --limit 5 | jq .
```

## Building from source

```sh
git clone https://github.com/unified-to/unified-cli.git
cd unified-cli
go build -o unified .
```

### Refreshing the object catalog

The object catalog in `internal/catalog/objects_gen.go` is generated from the
route definitions in the [unified-api](https://github.com/unified-to/unified-api)
repository. With a checkout of it alongside this one:

```sh
go generate ./internal/catalog/...
```

The generator cross-checks what it finds against the API's canonical
`ObjectType` list and fails if the two disagree, so an object added to the API
cannot go quietly missing from the CLI. See
[docs/plans/2026-09-17-object-catalog.md](docs/plans/2026-09-17-object-catalog.md).
