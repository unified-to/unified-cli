# Unified CLI Design

## Overview

A Go CLI tool for accessing the Unified.to API. Outputs raw JSON, works like curl, installable via Homebrew on Mac.

## Command Interface

```
unified <method> <connection_id> <object> [flags]
```

**Positional arguments:**
- `method`: one of `list`, `get`, `create`, `update`, `remove`
- `connection_id`: the Unified.to connection ID
- `object`: underscore-delimited object name (e.g., `ats_candidate`), split on first `_` into category + object for URL construction

**Examples:**
```
unified list abc123 ats_candidate
unified get abc123 ats_candidate --id ID456
unified create abc123 crm_contact -d '{"name":"John"}'
unified update abc123 crm_contact --id ID456 -d '{"name":"Jane"}'
unified remove abc123 crm_contact --id ID456
```

## Authentication

- Flag: `--api-key` / `-k`
- Env var: `UNIFIED_API_KEY`
- Flag takes precedence over env var
- Sent as `Authorization: bearer <API_KEY>` header

## Flags

### Global

| Flag | Short | Env Var | Required | Description |
|------|-------|---------|----------|-------------|
| `--api-key` | `-k` | `UNIFIED_API_KEY` | Yes | Bearer token |

### List-specific (known)

| Flag | Short | Description |
|------|-------|-------------|
| `--query` | `-q` | Search/filter query |
| `--limit` | `-l` | Max results |
| `--offset` | `-o` | Pagination offset |
| `--updated-gte` | | ISO-8601 filter |
| `--sort` | `-s` | Sort field |
| `--order` | | asc/desc |
| `--fields` | `-f` | Comma-delimited fields |

The `list` command also accepts arbitrary unknown flags (e.g., `--job_id J123`, `--type active`) which are passed verbatim as query parameters.

### Get / Update / Remove

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--id` | `-i` | Yes | Object ID |

### Create / Update

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--data` | `-d` | Yes | JSON body, inline or `@file.json` |

## API Mapping

Base URL: `https://api.unified.to`

Object parsing: `ats_candidate` splits on first `_` to category=`ats`, object=`candidate`.

| CLI Method | HTTP Method | URL Pattern |
|-----------|-------------|-------------|
| list | GET | `/{category}/{conn_id}/{object}` |
| get | GET | `/{category}/{conn_id}/{object}/{id}` |
| create | POST | `/{category}/{conn_id}/{object}` |
| update | PATCH | `/{category}/{conn_id}/{object}/{id}` |
| remove | DELETE | `/{category}/{conn_id}/{object}/{id}` |

## Output

- Success: raw JSON to stdout (pipeable to `jq`)
- Errors: message to stderr, exit code 1

## Error Handling

- Missing required args: Cobra prints usage + error, exit 1
- Missing `--id` for get/update/remove: error before HTTP call, exit 1
- Missing `--data` for create/update: error before HTTP call, exit 1
- HTTP errors: print status code + response body to stderr, exit 1
- Network errors: print error message to stderr, exit 1

## Project Structure

```
unified-cli/
├── main.go
├── go.mod / go.sum
├── cmd/
│   ├── root.go
│   ├── list.go
│   ├── get.go
│   ├── create.go
│   ├── update.go
│   └── remove.go
├── internal/
│   ├── api/
│   │   └── client.go
│   └── parser/
│       └── object.go
├── Formula/
│   └── unified.rb
├── .goreleaser.yaml
└── README.md
```

## Technology

- Language: Go
- CLI framework: Cobra
- HTTP: Go standard library `net/http`
- Build/release: GoReleaser
- Distribution: Homebrew formula (tap-ready)

## Build & Release

GoReleaser builds `darwin-arm64` and `darwin-amd64` binaries, generates tarballs with checksums, and produces a Homebrew formula. Mac only for now.

## Decisions

- Go with Cobra for CLI framework (mature ecosystem, shell completions, Homebrew-friendly)
- PATCH for updates (partial, safer for CLI use)
- Single combined object arg (`ats_candidate`) split on first `_`
- Connection ID as positional arg (required for every call)
- Unknown flags on `list` passed as verbatim query params
- API key via flag or env var, flag takes precedence
