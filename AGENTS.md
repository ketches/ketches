# Ketches

Ketches is a cloud-native application delivery platform. Users connect Kubernetes clusters, then deploy, operate, and collaborate on containerized applications through a unified web console.

- Backend (repo root): Go + Gin + GORM + client-go; API contract generated from `cmd/openapi`
- Frontend (`ui/`): React + TypeScript + Vite + Tailwind + shadcn/ui, with Zustand, TanStack Query, Axios
- Docs: design specs in `docs/specs/`, implementation plans in `docs/plans/`

## Repo layout

```txt
cmd/                    cmd/api (entrypoint), cmd/openapi (OpenAPI generator)
internal/routes         route registration -> internal/middlewares (JWT/RBAC)
internal/handlers       HTTP handlers -> internal/services (business logic)
internal/core           Kubernetes resource construction & reconciliation
internal/db             GORM entities (entities/) + migrations
internal/kube           cluster client management
internal/secrets        sensitive data handling (KubeConfigs, credentials)
pkg/                    shared helpers (concurrency, uuid, websocket, ...)
openapi/                openapi.yaml/json (GENERATED — never hand-edit)
ui/src/App.tsx          frontend routing (react-router)
ui/src/pages            application pages
ui/src/components/      feature components; ui/ subdir = shadcn components (FROZEN)
ui/src/api/             API client modules + generated/ TS types (GENERATED)
ui/src/stores           Zustand stores
docs/specs/             design docs (yyyy-mm-dd-{feat}.md)
docs/plans/             implementation plans (yyyy-mm-dd-{feat}.md)
```

## Commands

```txt
make run              # start backend locally (reads .env)
make dev-ui           # start Vite dev server for ui/
make test             # Go tests with race detector: go test -race ./...
cd ui && pnpm test    # frontend tests (vitest)
make lint             # golangci-lint on Go sources
cd ui && pnpm lint    # eslint
cd ui && pnpm build   # type-check (tsc -b) + vite build
make openapi          # regenerate openapi/* and ui/src/api/generated/* types
make build            # backend binary -> bin/ketches
```

## Architecture & data rules

- Request flow: `routes -> handlers -> services -> core` (Kubernetes) or `db` (persistence). Keep layering: handlers validate requests, services hold business logic, core owns resource construction.
- API changes start in `cmd/openapi`, then run `make openapi` to refresh the contract and TS types (single source of truth; treat generated files as build output).
- Entities stay scalar-only: no physical foreign keys in DDL/migrations, no GORM association tags (`foreignKey`, `references`, `constraint`). Resolve relationships via DTO/query models + `JOIN`. Legacy DBs: drop existing FKs with DDL scripts and normalize empty-string IDs to `NULL`.
- Sensitive data (KubeConfigs, registry credentials, passwords) must be handled through `internal/secrets` — never logged, echoed, or returned in API responses.

## Coding rules

### Backend

- English comments.
- Prefer `any` over `interface{}` for generic types.

### Frontend

- English comments; shadcn components only, no hand-written CSS (use Tailwind utilities).
- Never modify anything inside `ui/src/components/ui/`.
- Avoid the `any` type.
- All dropdowns/selection inputs use `Combobox` with const option arrays — never `Select`, and never use `value` as the displayed label:

```tsx
itemToStringLabel={(v) => FILE_MODE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
```

- Forms prefer `Field`, `FieldLabel`, `FieldContent`.
- Heights use `min-h-?` scale classes only (never `min-h-[?px]`); vertical spacing uses `space-y-?` (never `gap-y-?`).

### Docs

- Markdown lint: keep a space on both sides of table pipes; every code block declares a language (default `txt`).

## Docs & planning

Write docs for each feature and keep them updated in the same PR as the code:

- Design docs -> `docs/specs/yyyy-mm-dd-{feat}.md` — problem, goals, API/schema impact, alternatives.
- Implementation plans -> `docs/plans/yyyy-mm-dd-{feat}.md` — ordered steps, files to touch, test strategy.
- `yyyy-mm-dd` is the start date, `{feat}` a short kebab-case slug, e.g. `docs/specs/2025-06-01-app-gateway.md`.

## Git workflow

- Conventional Commits with optional scope — `cliff.toml` feeds the git-cliff changelog, so malformed subjects break release notes: `feat:`, `fix(ui):`, `docs:`, `refactor:`, `chore:`, `perf:`, `test:`.
- Use scope for subsystem hints: `ui`, `api`, `core`, `db`, `release`, `ci`.
- English, imperative subject; explain why in the body when non-obvious.

## Boundaries

Never do:

- Edit `ui/src/components/ui/*` or generated files (`openapi/*`, `ui/src/api/generated/*`, `bin/`, `dist/`) — regenerate instead.
- Add physical foreign keys or GORM association tags.
- Use `Select`, hand-written CSS, `min-h-[?px]`, or `gap-y-?`.
- Commit secrets, KubeConfigs, or `.env` values.

Ask first:

- DB schema changes that drop/rename columns; API contract changes; cross-module refactors.

## Agent roles

| Agent | Focus |
| --- | --- |
| System Architect | Cross-module consistency and integrations between backend and `ui/`. |
| Product & Requirements | Feature alignment with prioritized modules (P0/P1/P2); drives specs/plans docs. |
| Backend Engineer | RESTful APIs in `internal/handlers` + `internal/services`. |
| Kubernetes Specialist | `client-go` resources, reconciliation, and status logic in `internal/core`. |
| DB & ORM Specialist | Entities and migrations in `internal/db`. |
| Security & Auth | JWT, RBAC middlewares (AdminOnly, ProjectMember), secrets handling. |
| Frontend Engineer | Pages and feature components in `ui/src`. |
| UI/UX Design | Design-system consistency (shadcn/ui + Tailwind). |
| State & Data | Server-state and API client modules in `ui/src/api` + `ui/src/stores`. |
| DevOps UI | xterm.js terminals, WebSocket log streaming, real-time views. |
