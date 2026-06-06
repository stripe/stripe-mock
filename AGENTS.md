# Agents

This file gives AI coding agents (Cursor, Copilot, Claude Code, etc.) the context they need to work productively in this repo.

## What this project is
stripe-mock is a mock HTTP server that responds like the real Stripe API. It is used for integration testing in CI environments and local development. The tool is written in Go and simulates Stripe's API surface by parsing the OpenAPI specifications.

## Project layout
- `cmd/stripe-mock` — entry point for the executable.
- `spec/` — contains the OpenAPI specification files.
- `param/` — parameter parsing logic.
- `discovery/` — logic for discovering endpoints within the spec.

## How to build & test
```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build the binary
go build .
```

## Conventions
- Follow standard Go idioms and formatting (gofmt).
- Ensure new features align with the latest Stripe OpenAPI specification.
- Use explicit error handling over panics.

## Things to avoid
- Do not hardcode API responses that should be derived from the spec.
- Avoid adding dependencies that increase the binary size significantly.

## Where to look for help
- https://github.com/stripe/stripe-mock/issues
- https://stripe.com/docs/api
