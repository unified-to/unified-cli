# Unified CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool (`unified`) that wraps the Unified.to API with CRUD operations, outputting raw JSON.

**Architecture:** Cobra-based CLI with 5 subcommands (list, get, create, update, remove). Each subcommand validates its args, builds the API URL by splitting the object name (e.g., `ats_candidate` → `/ats/{conn}/candidate`), and delegates to a thin HTTP client. Output goes to stdout; errors to stderr.

**Tech Stack:** Go 1.23+, Cobra v1.9+, GoReleaser, Homebrew formula

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: Initialize Go module**

Run:
```bash
cd /Users/roypereira/Documents/dev/unified/unified-cli
go mod init github.com/unified-to/unified-cli
```

**Step 2: Install Cobra dependency**

Run:
```bash
go get github.com/spf13/cobra@latest
```

**Step 3: Create `main.go`**

```go
package main

import "github.com/unified-to/unified-cli/cmd"

func main() {
	cmd.Execute()
}
```

**Step 4: Create `cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiKey string

var rootCmd = &cobra.Command{
	Use:   "unified",
	Short: "CLI tool for the Unified.to API",
	Long:  "A command-line interface for accessing the Unified.to API.\nPerform CRUD operations on any Unified.to object.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "", "API key (or set UNIFIED_API_KEY env var)")
}

func getAPIKey() (string, error) {
	if apiKey != "" {
		return apiKey, nil
	}
	if envKey := os.Getenv("UNIFIED_API_KEY"); envKey != "" {
		return envKey, nil
	}
	return "", fmt.Errorf("API key required: use --api-key flag or set UNIFIED_API_KEY environment variable")
}
```

**Step 5: Verify it compiles**

Run: `go build -o unified .`
Expected: binary created, `./unified --help` shows usage

**Step 6: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: scaffold project with Cobra root command"
```

---

### Task 2: Object Parser

**Files:**
- Create: `internal/parser/object.go`
- Create: `internal/parser/object_test.go`

**Step 1: Write the failing test**

```go
package parser

import "testing"

func TestSplitObject(t *testing.T) {
	tests := []struct {
		input    string
		category string
		object   string
		wantErr  bool
	}{
		{"ats_candidate", "ats", "candidate", false},
		{"crm_contact", "crm", "contact", false},
		{"hris_employee", "hris", "employee", false},
		{"commerce_item", "commerce", "item", false},
		{"ats_application", "ats", "application", false},
		{"accounting_credit_memo", "accounting", "credit_memo", false},
		{"", "", "", true},
		{"nocategory", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			category, object, err := SplitObject(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SplitObject(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("SplitObject(%q) unexpected error: %v", tt.input, err)
				return
			}
			if category != tt.category {
				t.Errorf("SplitObject(%q) category = %q, want %q", tt.input, category, tt.category)
			}
			if object != tt.object {
				t.Errorf("SplitObject(%q) object = %q, want %q", tt.input, object, tt.object)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/parser/ -v`
Expected: FAIL (file doesn't exist yet)

**Step 3: Write minimal implementation**

```go
package parser

import (
	"fmt"
	"strings"
)

// SplitObject splits an object name like "ats_candidate" into category and object.
// Splits on the first underscore only, so "accounting_credit_memo" becomes
// category="accounting", object="credit_memo".
func SplitObject(input string) (category string, object string, err error) {
	if input == "" {
		return "", "", fmt.Errorf("object name cannot be empty")
	}
	idx := strings.Index(input, "_")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid object name %q: expected format like 'ats_candidate'", input)
	}
	return input[:idx], input[idx+1:], nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/parser/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parser/object.go internal/parser/object_test.go
git commit -m "feat: add object name parser with tests"
```

---

### Task 3: API Client

**Files:**
- Create: `internal/api/client.go`
- Create: `internal/api/client_test.go`

**Step 1: Write the failing test**

```go
package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/ats/conn123/candidate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)
	body, statusCode, err := c.Do("GET", "ats", "conn123", "candidate", "", nil, map[string]string{"limit": "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if strings.TrimSpace(string(body)) != `[{"id":"1"}]` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/conn456/contact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		reqBody, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(reqBody)) != `{"name":"John"}` {
			t.Errorf("unexpected body: %s", reqBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"new1","name":"John"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)
	data := []byte(`{"name":"John"}`)
	body, statusCode, err := c.Do("POST", "crm", "conn456", "contact", "", data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 201 {
		t.Errorf("expected status 201, got %d", statusCode)
	}
	if !strings.Contains(string(body), `"name":"John"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientGetWithID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ats/conn123/candidate/id789" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":"id789"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)
	body, _, err := c.Do("GET", "ats", "conn123", "candidate", "id789", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), `"id789"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)
	body, statusCode, err := c.Do("GET", "ats", "conn123", "candidate", "bad", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 404 {
		t.Errorf("expected status 404, got %d", statusCode)
	}
	if !strings.Contains(string(body), "not found") {
		t.Errorf("unexpected body: %s", body)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const DefaultBaseURL = "https://api.unified.to"

// Client is an HTTP client for the Unified.to API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new API client. If baseURL is empty, uses the default.
func NewClient(apiKey string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

// Do performs an HTTP request against the Unified.to API.
// Returns the response body, status code, and any error.
func (c *Client) Do(method, category, connectionID, object, id string, body []byte, params map[string]string) ([]byte, int, error) {
	// Build URL path
	path := fmt.Sprintf("/%s/%s/%s", category, connectionID, object)
	if id != "" {
		path = fmt.Sprintf("%s/%s", path, id)
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Build request
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat: add API client with tests"
```

---

### Task 4: List Subcommand

**Files:**
- Create: `cmd/list.go`

**Step 1: Implement the list command**

This is the most complex subcommand due to arbitrary unknown flags. We use `cobra.Command.FParseErrWhitelist` to allow unknown flags and parse them manually from `os.Args`.

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var listQuery string
var listLimit string
var listOffset string
var listUpdatedGte string
var listSort string
var listOrder string
var listFields string

var listCmd = &cobra.Command{
	Use:   "list <connection_id> <object>",
	Short: "List objects",
	Long:  "List objects from the Unified.to API.\nExample: unified list abc123 ats_candidate --limit 10 --job_id J123",
	Args:  cobra.ExactArgs(2),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		// Build query params from known flags
		params := map[string]string{}
		if listQuery != "" {
			params["query"] = listQuery
		}
		if listLimit != "" {
			params["limit"] = listLimit
		}
		if listOffset != "" {
			params["offset"] = listOffset
		}
		if listUpdatedGte != "" {
			params["updated_gte"] = listUpdatedGte
		}
		if listSort != "" {
			params["sort"] = listSort
		}
		if listOrder != "" {
			params["order"] = listOrder
		}
		if listFields != "" {
			params["fields"] = listFields
		}

		// Parse unknown flags as extra query params
		extraParams := parseUnknownFlags(os.Args)
		for k, v := range extraParams {
			params[k] = v
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("GET", category, connectionID, object, "", nil, params)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		fmt.Print(string(body))
		return nil
	},
}

// parseUnknownFlags extracts --key value pairs that aren't known Cobra flags.
func parseUnknownFlags(args []string) map[string]string {
	knownFlags := map[string]bool{
		"--api-key": true, "-k": true,
		"--query": true, "-q": true,
		"--limit": true, "-l": true,
		"--offset": true, "-o": true,
		"--updated-gte": true,
		"--sort": true, "-s": true,
		"--order": true,
		"--fields": true, "-f": true,
		"--help": true, "-h": true,
	}

	extra := map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		// Handle --key=value
		if idx := strings.Index(arg, "="); idx != -1 {
			key := arg[:idx]
			if !knownFlags[key] {
				extra[strings.TrimPrefix(key, "--")] = arg[idx+1:]
			}
			continue
		}
		// Handle --key value
		if !knownFlags[arg] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			extra[strings.TrimPrefix(arg, "--")] = args[i+1]
			i++
		}
	}
	return extra
}

func init() {
	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Search/filter query")
	listCmd.Flags().StringVarP(&listLimit, "limit", "l", "", "Max results to return")
	listCmd.Flags().StringVarP(&listOffset, "offset", "o", "", "Pagination offset")
	listCmd.Flags().StringVar(&listUpdatedGte, "updated-gte", "", "ISO-8601 datetime filter")
	listCmd.Flags().StringVarP(&listSort, "sort", "s", "", "Sort field")
	listCmd.Flags().StringVar(&listOrder, "order", "", "Sort direction (asc/desc)")
	listCmd.Flags().StringVarP(&listFields, "fields", "f", "", "Comma-delimited fields to return")
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified .`
Expected: builds cleanly, `./unified list --help` shows usage

**Step 3: Commit**

```bash
git add cmd/list.go
git commit -m "feat: add list subcommand with arbitrary query param support"
```

---

### Task 5: Get Subcommand

**Files:**
- Create: `cmd/get.go`

**Step 1: Implement the get command**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var getID string

var getCmd = &cobra.Command{
	Use:   "get <connection_id> <object>",
	Short: "Get a single object by ID",
	Long:  "Get a single object from the Unified.to API.\nExample: unified get abc123 ats_candidate --id ID456",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if getID == "" {
			return fmt.Errorf("--id is required for get")
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("GET", category, connectionID, object, getID, nil, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		fmt.Print(string(body))
		return nil
	},
}

func init() {
	getCmd.Flags().StringVarP(&getID, "id", "i", "", "Object ID (required)")
	getCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified . && ./unified get --help`
Expected: shows usage with --id flag

**Step 3: Commit**

```bash
git add cmd/get.go
git commit -m "feat: add get subcommand"
```

---

### Task 6: Create Subcommand

**Files:**
- Create: `cmd/create.go`

**Step 1: Implement the create command**

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var createData string

var createCmd = &cobra.Command{
	Use:   "create <connection_id> <object>",
	Short: "Create a new object",
	Long:  "Create a new object via the Unified.to API.\nExample: unified create abc123 crm_contact -d '{\"name\":\"John\"}'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if createData == "" {
			return fmt.Errorf("--data is required for create")
		}

		data, err := readData(createData)
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("POST", category, connectionID, object, "", data, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		fmt.Print(string(body))
		return nil
	},
}

// readData returns the JSON data from the flag value.
// If the value starts with @, it reads the file at that path.
func readData(value string) ([]byte, error) {
	if strings.HasPrefix(value, "@") {
		filePath := strings.TrimPrefix(value, "@")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read data file %q: %w", filePath, err)
		}
		return data, nil
	}
	return []byte(value), nil
}

func init() {
	createCmd.Flags().StringVarP(&createData, "data", "d", "", "JSON data (inline or @file.json) (required)")
	createCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(createCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified . && ./unified create --help`
Expected: shows usage with --data flag

**Step 3: Commit**

```bash
git add cmd/create.go
git commit -m "feat: add create subcommand with @file.json support"
```

---

### Task 7: Update Subcommand

**Files:**
- Create: `cmd/update.go`

**Step 1: Implement the update command**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var updateID string
var updateData string

var updateCmd = &cobra.Command{
	Use:   "update <connection_id> <object>",
	Short: "Update an existing object (partial update via PATCH)",
	Long:  "Update an existing object via the Unified.to API using PATCH.\nExample: unified update abc123 crm_contact --id ID456 -d '{\"name\":\"Jane\"}'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if updateID == "" {
			return fmt.Errorf("--id is required for update")
		}
		if updateData == "" {
			return fmt.Errorf("--data is required for update")
		}

		data, err := readData(updateData)
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("PATCH", category, connectionID, object, updateID, data, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		fmt.Print(string(body))
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateID, "id", "i", "", "Object ID (required)")
	updateCmd.Flags().StringVarP(&updateData, "data", "d", "", "JSON data (inline or @file.json) (required)")
	updateCmd.MarkFlagRequired("id")
	updateCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(updateCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified . && ./unified update --help`
Expected: shows usage with --id and --data flags

**Step 3: Commit**

```bash
git add cmd/update.go
git commit -m "feat: add update subcommand"
```

---

### Task 8: Remove Subcommand

**Files:**
- Create: `cmd/remove.go`

**Step 1: Implement the remove command**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var removeID string

var removeCmd = &cobra.Command{
	Use:   "remove <connection_id> <object>",
	Short: "Remove an object by ID",
	Long:  "Remove an object from the Unified.to API.\nExample: unified remove abc123 crm_contact --id ID456",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if removeID == "" {
			return fmt.Errorf("--id is required for remove")
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("DELETE", category, connectionID, object, removeID, nil, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		if len(body) > 0 {
			fmt.Print(string(body))
		}
		return nil
	},
}

func init() {
	removeCmd.Flags().StringVarP(&removeID, "id", "i", "", "Object ID (required)")
	removeCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(removeCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified . && ./unified remove --help`
Expected: shows usage with --id flag

**Step 3: Commit**

```bash
git add cmd/remove.go
git commit -m "feat: add remove subcommand"
```

---

### Task 9: Version Command & Build Info

**Files:**
- Modify: `main.go`
- Create: `cmd/version.go`

**Step 1: Add version command**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set by GoReleaser via ldflags
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build -o unified . && ./unified version`
Expected: prints `dev`

**Step 3: Commit**

```bash
git add cmd/version.go
git commit -m "feat: add version subcommand"
```

---

### Task 10: GoReleaser & Homebrew Formula

**Files:**
- Create: `.goreleaser.yaml`
- Create: `.gitignore`

**Step 1: Create `.gitignore`**

```
unified
dist/
```

**Step 2: Create `.goreleaser.yaml`**

```yaml
version: 2

builds:
  - main: .
    binary: unified
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/unified-to/unified-cli/cmd.Version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

brews:
  - name: unified
    repository:
      owner: unified-to
      name: homebrew-cli
    directory: Formula
    homepage: "https://unified.to"
    description: "CLI tool for the Unified.to API"
    license: "MIT"
    install: |
      bin.install "unified"
    test: |
      assert_match "unified", shell_output("#{bin}/unified --help")

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

**Step 3: Create standalone Homebrew formula template**

Create `Formula/unified.rb`:

```ruby
class Unified < Formula
  desc "CLI tool for the Unified.to API"
  homepage "https://unified.to"
  license "MIT"

  # Updated by GoReleaser
  url "https://github.com/unified-to/unified-cli/releases/download/v0.0.0/unified_0.0.0_darwin_arm64.tar.gz"
  sha256 "PLACEHOLDER"

  def install
    bin.install "unified"
  end

  test do
    assert_match "unified", shell_output("#{bin}/unified --help")
  end
end
```

**Step 4: Commit**

```bash
git add .goreleaser.yaml .gitignore Formula/unified.rb
git commit -m "feat: add GoReleaser config and Homebrew formula"
```

---

### Task 11: End-to-End Smoke Test

**Files:**
- None (manual verification)

**Step 1: Build and test help output**

Run:
```bash
go build -o unified .
./unified --help
./unified list --help
./unified get --help
./unified create --help
./unified update --help
./unified remove --help
./unified version
```

Expected: all commands show proper help text, version prints `dev`

**Step 2: Run all unit tests**

Run: `go test ./... -v`
Expected: all tests pass

**Step 3: Final commit and clean up**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: tidy go modules"
```

---

### Task 12: README

**Files:**
- Create: `README.md`

**Step 1: Write README with install, usage, and examples**

Cover: what it is, how to install (Homebrew + `go install`), usage examples for all 5 methods, environment variable config, and building from source.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with install and usage instructions"
```
