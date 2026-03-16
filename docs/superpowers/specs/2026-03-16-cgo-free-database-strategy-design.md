# CGO-Free Database Strategy Design

## Summary

Ketches should stop treating SQLite as a first-class runtime database and move to a CGO-free deployment path. The selected design is:

- runtime and local development use PostgreSQL as the primary supported database
- tests keep SQLite for speed and isolation, but switch to a pure-Go GORM-compatible driver
- Docker and local build targets switch to `CGO_ENABLED=0`

This removes the current CGO requirement that comes from the SQLite runtime driver while keeping test feedback fast.

## Goals

- Remove CGO from the backend build and Docker image.
- Make PostgreSQL the default and recommended database for runtime use.
- Keep the codebase easy to test without requiring a database container for every unit-style test.
- Minimize the migration surface so the change can land without redesigning the entire persistence layer.

## Non-Goals

- Removing all non-PostgreSQL code paths in one pass.
- Building a production-grade SQLite runtime path.
- Replacing GORM or rewriting database access patterns.
- Converting all tests to PostgreSQL-backed integration tests.

## Current Problem

Today the backend statically imports `gorm.io/driver/sqlite` in [internal/db/db.go](/Users/dp/ketches/.worktrees/cgo-free-postgres-default/internal/db/db.go#L8), keeps `sqlite` as the default `DB_DRIVER` in [internal/app/config.go](/Users/dp/ketches/.worktrees/cgo-free-postgres-default/internal/app/config.go#L31), and builds Docker with `CGO_ENABLED=1` in [Dockerfile](/Users/dp/ketches/.worktrees/cgo-free-postgres-default/Dockerfile#L22). The CGO dependency is therefore not incidental; it is part of the default runtime path.

This creates three concrete problems:

- container builds need extra toolchain packages only for SQLite linkage
- development defaults do not match the recommended production database path
- the project continues to validate runtime behavior against a database that is explicitly not the preferred deployment choice

## Alternatives Considered

### Option 1: Keep SQLite runtime support and switch to a pure-Go SQLite driver everywhere

This removes CGO, but it preserves the deeper issue that runtime behavior is still split between SQLite and PostgreSQL. It would improve build ergonomics while keeping the product and test surface centered on a database that should not be the default runtime path.

### Option 2: Use PostgreSQL for runtime and pure-Go SQLite only for tests

This is the chosen approach.

Why it wins:

- removes the CGO dependency from production and development builds
- aligns runtime defaults with the recommended database
- keeps unit-style tests fast and cheap
- limits application code changes to configuration, driver wiring, docs, and tests

### Option 3: Use PostgreSQL for runtime, development, and all tests

This gives the strongest behavioral consistency, but it raises the cost of every test run and would require broader test harness changes. That is too heavy for this iteration.

## Chosen Architecture

The project will distinguish clearly between runtime database support and test-only database support.

### Runtime Database Policy

Runtime support becomes PostgreSQL-first:

- `postgres` is the default `DB_DRIVER`
- PostgreSQL is the documented and recommended deployment path
- SQLite is removed from the runtime initialization switch

MySQL support can remain in code for now because it does not drive the CGO problem, but it is not the primary validated path for this change.

### Test Database Policy

Tests that currently use in-memory SQLite continue to do so, but they switch from `gorm.io/driver/sqlite` to a pure-Go GORM-compatible driver. This keeps the existing lightweight test setup while removing the `github.com/mattn/go-sqlite3` CGO dependency from the module graph.

The preferred test driver is `github.com/glebarez/sqlite` because it is GORM-compatible with minimal call-site change.

### Build and Packaging Policy

Once runtime SQLite is removed and tests no longer import the CGO-backed driver:

- Docker builds use `CGO_ENABLED=0`
- GCC and musl build dependencies are removed from the builder image
- local backend build targets also use `CGO_ENABLED=0`

This makes the backend binary easier to build across environments and reduces the Docker build surface.

## Implementation Scope

### Runtime Code Changes

Update [internal/db/db.go](/Users/dp/ketches/.worktrees/cgo-free-postgres-default/internal/db/db.go#L16) to stop importing and opening SQLite at runtime. The supported runtime switch becomes:

- `postgres`
- `mysql`

Any runtime attempt to set `DB_DRIVER=sqlite` should now fail fast with a clear unsupported-driver error.

### Configuration Changes

Update [internal/app/config.go](/Users/dp/ketches/.worktrees/cgo-free-postgres-default/internal/app/config.go#L30) and related config tests so the default `DB_DRIVER` becomes `postgres`. The split-variable connection builder should continue to support:

- PostgreSQL defaults for runtime
- MySQL as an optional alternative path

The SQLite-specific fallback behavior for `DB_NAME` should be removed from the default branch logic.

### Test Changes

All tests importing `gorm.io/driver/sqlite` should move to the pure-Go driver. The helper calls remain effectively the same:

```go
testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{...})
```

The intent is to preserve test semantics while changing only the driver package.

### Documentation Changes

Update README, Chinese README, Docker comments, and any environment examples so they reflect:

- PostgreSQL as the default database
- SQLite no longer being part of the supported runtime path
- CGO no longer being required for backend builds

## Risks and Mitigations

### Risk 1: Some tests rely on SQLite-specific behavior

Even when both drivers target SQLite semantics, different driver implementations can expose timing or locking differences.

Mitigation:

- migrate tests incrementally
- run the full backend suite after the swap
- keep changes minimal and avoid changing test intent at the same time

### Risk 2: Existing users may still rely on runtime SQLite

Removing runtime SQLite is a breaking behavior change for anyone depending on it.

Mitigation:

- update documentation clearly
- return a direct unsupported-driver error instead of silently falling back
- call out the change in release notes or migration notes if those are maintained

### Risk 3: PostgreSQL default may affect developer onboarding

Changing the default runtime database means local startup now requires a PostgreSQL instance or explicit env override to MySQL.

Mitigation:

- keep Docker Compose or Helm examples easy to use
- document a minimal PostgreSQL local setup path

## Testing Strategy

Verification should cover:

- config tests proving `postgres` is now the default driver
- runtime DB init tests proving `sqlite` is rejected
- focused tests proving the pure-Go SQLite driver still supports current in-memory test setup
- full backend test suite
- backend build with `CGO_ENABLED=0`
- Docker image build without GCC toolchain packages

## Rollout

This can land as a single focused change because the codebase already supports PostgreSQL. No data migration is needed inside the project itself; the change is about supported runtime drivers and build strategy rather than schema design.
