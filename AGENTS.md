# Repository Guidelines

## Project Overview
`yo` is a command-line tool designed to generate Go code for [Google Cloud Spanner](https://cloud.google.com/spanner/). It parses a Spanner database schema (retrieved either from a live database's `INFORMATION_SCHEMA` or from a local DDL file) and uses Go `text/template` files to generate type-safe Go models, mutation wrappers (`Insert`, `Update`), and query/read functions.

## Architecture & Data Flow
The code generation process follows a structured pipeline:
1.  **Loading**: The `loader` (legacy `loaders/` or refactored `v2/loader/`) parses the schema source (DDL or live Spanner instance) and maps it into an internal representation.
2.  **Intermediate Representation (IR)**: The schema is represented by structs defined in `models/` (e.g., `Table`, `Column`, `Index`), which serve as the source of truth for the generator.
3.  **Generation**: The `generator/` applies `text/template` definitions found in `templates/` to the IR to produce the final Go code.
4.  **Embedding**: Templates are compiled into Go code using `go-assets-builder` and placed in `tplbin/` for embedding into the final binary.

The `v2/` directory represents a significant refactor aimed at decoupling schema extraction from model building and code generation.

## Key Directories
- `loaders/` / `v2/loader/`: Logic for schema parsing and extraction.
- `models/` / `v2/models/`: Internal schema representation structs.
- `generator/` / `v2/generator/`: Logic for applying templates to IR models.
- `templates/`: Source `text/template` files used for code generation.
- `tplbin/`: Generated, embedded template Go code.
- `test/`: Integration tests and test data.

## Development Commands
- **Build**: `make build` (triggers `regen` + `go build`).
- **Regenerate Templates**: `make regen` (compiles `templates/` to `tplbin/templates.go`).
- **Run Tests**: `make test` (automatically spins up a ephemeral Spanner emulator via Docker).
- **Run Lint**: `make lint` (`go fmt` + `go vet`).

## Code Conventions & Common Patterns
- **Templates**: Uses standard `text/template`.
- **Error Handling**: Standard Go error patterns; `yo` defines custom error types for Spanner interactions (e.g., `yoError` implementing `GRPCStatus`).
- **Templates Generation**: `tplbin/templates.go` is machine-generated; do not edit it directly. Update the source files in `templates/` and run `make regen`.

## Important Files
- `Makefile`: Central orchestrator for building, testing (including emulator lifecycle), and template regeneration.
- `main.go`: Entry point for the `yo` CLI.
- `internal/`: Shared utilities and internal types.

## Runtime/Tooling Preferences
- **Language**: Go.
- **Dependency Management**: Go modules.
- **Tooling**: Requires Docker for running integration tests (Spanner emulator). Requires `github.com/jessevdk/go-assets-builder` for template compilation.

## Testing & QA
- Integration tests in `test/integration_test.go` are the primary source of truth.
- Tests depend on a running Spanner emulator. The `Makefile` manages the emulator lifecycle (`_spanner-up`, `_spanner-down`) when `make test` is executed.
- Schema metadata correctness is verified against generated models within these integration tests.
