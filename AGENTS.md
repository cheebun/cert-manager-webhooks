# AGENTS.md

Instructions for AI-assisted development on this repository.

## Project Overview

Monorepo of cert-manager DNS01 webhook solvers. Each provider is an independent Go module
under `webhooks/`, sharing a common bootstrap library in `pkg/webhook/`.

## Repository Structure

```
pkg/webhook/              → Shared server library (Go module)
webhooks/alidns/           → Alibaba Cloud DNS solver (Go module)
webhooks/namesilo/         → NameSilo DNS solver (Go module)
webhooks/spaceship/        → Spaceship DNS solver (Go module)
webhooks/tencent/          → Tencent Cloud DNS solver (Go module)
```

## Go Module Layout

- Root `go.work` manages the workspace. **Do not delete or bypass it.**
- Each directory with a `go.mod` is an independent module.
- `pkg/webhook` is referenced via `go.work` locally; in CI it is fetched by version tag.
- When adding a dependency, add it to the correct module's `go.mod`, not to another module.

## Coding Conventions

### Language

- All code, comments, variable names, commit messages, and documentation: **English only**.

### Go Standards

- Follow [Effective Go](https://go.dev/doc/effective_go) and
  [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- Error handling: **always wrap errors** with `fmt.Errorf("context: %w", err)`.
- No `panic` in library code (`pkg/webhook`). Solvers may only panic on truly unrecoverable
  init-time errors.
- Logging: use `klog` (consistent with cert-manager ecosystem).

### Solver Implementation

Every solver must implement the `webhook.Solver` interface:

```go
type Solver interface {
    Name() string
    Present(ch *v1alpha1.ChallengeRequest) error
    CleanUp(ch *v1alpha1.ChallengeRequest) error
    Initialize(kubeClientSet kubernetes.Interface, stopCh <-chan struct{}) error
}
```

Rules for solver implementations:

1. **All DNS API calls must have timeouts.** Use `context.WithTimeout` — never rely on
   the client library's default (which may be infinite).
2. **Retry transient failures** (HTTP 429, 500, 502, 503, 504) with exponential backoff.
   Use `pkg/webhook` retry helpers.
3. **CleanUp must be idempotent.** If the TXT record does not exist, return `nil`, not an error.
4. **Credentials are read from Kubernetes Secrets**, not from environment variables or files.
   The Secret reference comes from the solver config JSON in the ChallengeRequest.
5. **Never log credentials.** Log the zone/domain being operated on, never the API key/secret.

### Helm Charts

- One chart per provider at `webhooks/<provider>/deploy/chart/`.
- Chart name: `cert-manager-webhook-<provider>`.
- Default `groupName` in `values.yaml` must be a placeholder (e.g., `acme.example.com`) —
  the user must set it explicitly.
- RBAC: request only the minimum permissions needed (Secrets read in the release namespace).

### Dockerfile

- Multi-stage build: builder stage compiles the Go binary, runtime stage uses `gcr.io/distroless/static`.
- Build context is the specific provider directory, not the repo root.
- The binary must be statically linked (`CGO_ENABLED=0`).

## Testing

- Unit tests: mock the DNS provider API. Use `httptest.NewServer` for HTTP-based APIs.
- Integration tests: gated behind build tag `// go:build integration`.
- Test file naming: `solver_test.go`, not `solver_unit_test.go`.

## Commit Conventions

```
<type>(<scope>): <description>

type:  feat | fix | refactor | docs | test | ci | chore
scope: alidns | namesilo | spaceship | tencent | webhook | repo
```

Examples:
- `feat(alidns): add support for VPC internal zones`
- `fix(webhook): handle graceful shutdown on SIGTERM`
- `docs(repo): update architecture diagram`

## CI / Build

- `make build PROVIDER=<name>` builds one provider.
- `make build-all` builds all providers.
- `make test` runs unit tests across all modules.
- `make lint` runs `golangci-lint` across all modules.

## Adding a New Provider

1. Copy an existing provider directory (e.g., `webhooks/alidns/`) as a template.
2. Update `go.mod` module path.
3. Implement `solver.go` with the provider's DNS API.
4. Add the module to `go.work`.
5. Create a Helm chart under `deploy/chart/`.
6. Add CI entries for the new provider.
7. Update root `README.md` provider table.
