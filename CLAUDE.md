# CLAUDE.md

This file provides guidance to Claude Code when working on this project.

@AGENTS.md

## Claude Code Specific

### Build & Test Commands

```bash
# Build a single provider
cd webhooks/<provider> && go build -o webhook .

# Test a single provider
cd webhooks/<provider> && go test ./...

# Test all modules via workspace
go test ./...

# Lint
golangci-lint run ./...
```

### Working with Go Workspace

This repo uses `go.work`. When modifying `pkg/webhook`, all provider modules pick up
changes immediately without version bumps. Run `go work sync` after adding new modules.

### File Ownership

When editing files, stay within one module per task:
- `pkg/webhook/*` — shared server logic only
- `webhooks/<provider>/*` — provider-specific logic only
- Never import one provider from another provider
