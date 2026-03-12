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

### List objects

```sh
unified list abc123 ats_candidate --limit 10
```

Any unknown flags are forwarded as query parameters:

```sh
unified list abc123 ats_candidate --limit 10 --job_id J123
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

## Object naming

Object names use the format `category_objectname`. The CLI splits on the first underscore to derive the API category and object:

- `ats_candidate` -> category `ats`, object `candidate`
- `crm_contact` -> category `crm`, object `contact`
- `accounting_credit_memo` -> category `accounting`, object `credit_memo`

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
