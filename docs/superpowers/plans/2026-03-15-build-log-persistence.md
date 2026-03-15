# Build Log Persistence Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist completed build logs to API-mounted storage so `succeeded`, `failed`, and `cancelled` builds remain viewable after the Kubernetes build Pod is deleted.

**Architecture:** Keep active build logs on the existing Kubernetes SSE path, but archive logs once at terminal state into files under a persistent API-local directory. Put archive/write/read/cleanup primitives in `internal/core` so `BuildWatcher`, cancellation, and startup recovery can share them without introducing package cycles; keep HTTP routing and SSE response formatting in `internal/services`.

**Tech Stack:** Go 1.25, Gin, GORM, client-go, Kubernetes fake clientsets in tests, Docker Compose, Helm, React 19, TypeScript, TanStack Query, Vitest

---

## Preconditions

- Execute this plan in a dedicated worktree. The current `/Users/dp/ketches` checkout is dirty and should not be used for implementation work.
- Before writing implementation code, invoke `@superpowers/test-driven-development`.
- Before claiming the work is done, invoke `@superpowers/verification-before-completion`.

## File Map

### Backend

- Modify: `internal/db/entities/build.go`
  - Add build log archive metadata columns to the `Build` entity.
- Modify: `internal/models/build.go`
  - Expose archive status metadata on `BuildResponse` so the UI can distinguish archive failures and expired logs.
- Modify: `internal/app/config.go`
  - Add `BUILD_LOG_BASE_DIR` and `BUILD_LOG_RETENTION_DAYS` parsing.
- Modify: `internal/app/config_test.go`
  - Lock config defaults and env overrides for build log settings.
- Create: `internal/core/build_log_archive.go`
  - Own terminal log archive path generation, pod lookup, log collection, temporary-file writes, atomic rename, archive metadata persistence, and archived-file reads.
- Create: `internal/core/build_log_archive_test.go`
  - Unit tests for archive writing order, idempotency, temp-file handling, and metadata updates using fake Kubernetes clients and temp dirs.
- Create: `internal/core/build_log_maintenance.go`
  - Recover missed terminal archives at API boot and clean expired archives on a timer.
- Create: `internal/core/build_log_maintenance_test.go`
  - Unit tests for restart recovery selection and expired-file cleanup behavior.
- Modify: `internal/core/build_watcher.go`
  - Persist logs on successful and failed build completion before watcher exit.
- Create: `internal/services/build_logs.go`
  - Keep `StreamBuildLogs` focused on status-based read routing and SSE formatting instead of growing `internal/services/build.go`.
- Create: `internal/services/build_logs_test.go`
  - Handler-level tests for active-build Pod streaming, terminal archived-file streaming, and missing-log error behavior.
- Modify: `internal/services/build.go`
  - Update `CancelBuild`, `ToBuildResponse`, and any helpers needed to integrate archive metadata and terminal log persistence.
- Modify: `cmd/api/main.go`
  - Start archive recovery and cleanup loops during API boot after cluster initialization.

### Deployment

- Modify: `deploy/docker/docker-compose.yml`
  - Mount a persistent build-log volume into the API container and pass build-log env vars.
- Modify: `deploy/helm/ketches/values.yaml`
  - Add API build-log persistence settings and env defaults.
- Modify: `deploy/helm/ketches/templates/api-deployment.yaml.tpl`
  - Mount the API build-log volume and inject archive env vars.
- Create: `deploy/helm/ketches/templates/api-pvc.yaml.tpl`
  - Define a PVC for API build-log storage when persistence is enabled.
- Modify: `deploy/helm/ketches/README.md`
  - Document the new build-log persistence values and the single-replica assumption.

### Frontend

- Modify: `ui/src/api/builds.ts`
  - Add archive status and error fields to the shared `Build` type.
- Modify: `ui/src/components/builds/build-log-viewer.tsx`
  - Surface archive failure and archive-expired states using data the view already fetches.
- Create: `ui/src/components/builds/build-log-viewer.test.tsx`
  - Verify the viewer shows archive-state messaging without breaking the existing log stream rendering path.

## Chunk 1: Archive Metadata and Core Persistence Primitives

### Task 1: Lock config and build-response archive metadata with failing tests

**Files:**
- Modify: `internal/models/build.go`
- Modify: `internal/app/config.go`
- Modify: `internal/app/config_test.go`
- Test: `internal/app/config_test.go`

- [ ] **Step 1: Add failing config tests for build-log defaults and env overrides**

```go
func TestInitConfig_DefaultsBuildLogSettings(t *testing.T)
func TestInitConfig_UsesBuildLogEnvOverrides(t *testing.T)
```

The assertions should cover:

- default `BUILD_LOG_BASE_DIR` resolves to `data/build-logs`
- default `BUILD_LOG_RETENTION_DAYS` resolves to `15`
- explicit env vars override both values

- [ ] **Step 2: Run the focused config tests to verify they fail**

Run: `go test ./internal/app -run 'TestInitConfig_(DefaultsBuildLogSettings|UsesBuildLogEnvOverrides)'`
Expected: FAIL because `AppConfig` does not yet expose build-log fields.

- [ ] **Step 3: Add the new config fields and build-response metadata**

```go
type AppConfig struct {
    // existing fields...
    BuildLogBaseDir       string
    BuildLogRetentionDays int
}
```

```go
type BuildResponse struct {
    // existing fields...
    LogPersistStatus string `json:"log_persist_status"`
    LogPersistError  string `json:"log_persist_error"`
}
```

Use `data/build-logs` and `15` as defaults.

- [ ] **Step 4: Re-run the focused config tests**

Run: `go test ./internal/app -run 'TestInitConfig_(DefaultsBuildLogSettings|UsesBuildLogEnvOverrides)'`
Expected: PASS

- [ ] **Step 5: Commit the config-and-contract slice**

```bash
git add internal/models/build.go internal/app/config.go internal/app/config_test.go
git commit -m "feat: add build log archive config and response metadata"
```

### Task 2: Lock archive persistence behavior with failing core tests

**Files:**
- Modify: `internal/db/entities/build.go`
- Create: `internal/core/build_log_archive.go`
- Create: `internal/core/build_log_archive_test.go`
- Test: `internal/core/build_log_archive_test.go`

- [ ] **Step 1: Write the failing archive tests**

```go
func TestPersistBuildLogs_WritesContainersInDisplayOrder(t *testing.T)
func TestPersistBuildLogs_IsIdempotentWhenArchiveAlreadyExists(t *testing.T)
func TestPersistBuildLogs_UsesTemporaryFileThenRenames(t *testing.T)
func TestPersistBuildLogs_MarksFailureWhenPodLogsUnavailable(t *testing.T)
func TestOpenPersistedBuildLog_FailsWhenArchiveMissing(t *testing.T)
```

Test setup should:

- create a temp archive directory
- seed a `builds` row and related `envs` row in sqlite
- use a fake Kubernetes clientset with a Pod labeled `job-name=<job>`
- return different logs for `git-clone` and `buildctl`

- [ ] **Step 2: Run the focused core tests to verify they fail**

Run: `go test ./internal/core -run 'TestPersistBuildLogs|TestOpenPersistedBuildLog'`
Expected: FAIL because archive helpers and entity fields do not exist yet.

- [ ] **Step 3: Implement build archive metadata and core persistence helpers**

Add build entity fields:

```go
LogPath         string `gorm:"type:varchar(512)"`
LogSize         int64  `gorm:"type:bigint"`
LogPersistStatus string `gorm:"type:varchar(32);default:'pending'"`
LogPersistError string `gorm:"type:text"`
LogPersistedAt  *time.Time
LogExpireAt     *time.Time
```

Use these persisted status values:

- `pending`
- `succeeded`
- `failed`
- `expired`

Implement focused helpers in `internal/core/build_log_archive.go`:

```go
func PersistBuildLogs(ctx context.Context, buildID string) error
func OpenPersistedBuildLog(build *entities.Build) (io.ReadCloser, error)
func BuildHasPersistedLog(build *entities.Build) bool
```

Use a private helper that accepts `kubernetes.Interface` so tests can use `fake.NewSimpleClientset`.

- [ ] **Step 4: Re-run the focused core tests**

Run: `go test ./internal/core -run 'TestPersistBuildLogs|TestOpenPersistedBuildLog'`
Expected: PASS

- [ ] **Step 5: Commit the archive core slice**

```bash
git add internal/db/entities/build.go internal/core/build_log_archive.go internal/core/build_log_archive_test.go
git commit -m "feat: add core build log archive persistence"
```

## Chunk 2: Terminal-State Integration and SSE Read Routing

### Task 3: Lock terminal archive integration in watcher and cancellation flows

**Files:**
- Modify: `internal/core/build_watcher.go`
- Modify: `internal/services/build.go`
- Modify: `internal/models/build.go`
- Create: `internal/services/build_logs.go`
- Create: `internal/services/build_logs_test.go`
- Test: `internal/core/build_watcher_test.go`
- Test: `internal/services/build_logs_test.go`

- [ ] **Step 1: Add failing tests for terminal archive integration**

Add watcher tests:

```go
func TestUpdateBuildFailed_PreservesArchiveFailureMetadata(t *testing.T)
```

Add service tests:

```go
func TestStreamBuildLogs_StreamsActivePodLogs(t *testing.T)
func TestStreamBuildLogs_StreamsArchivedLogsForTerminalBuild(t *testing.T)
func TestStreamBuildLogs_ReturnsErrorWhenTerminalArchiveMissing(t *testing.T)
func TestCancelBuild_PersistsAvailableLogsBeforeDeletingJob(t *testing.T)
```

In service tests, use `httptest.NewRecorder()` and a Gin test context to assert SSE output and HTTP status codes.

- [ ] **Step 2: Run the focused watcher and service tests to verify they fail**

Run: `go test ./internal/core -run 'TestUpdateBuildFailed_PreservesArchiveFailureMetadata' && go test ./internal/services -run 'Test(StreamBuildLogs|CancelBuild)'`
Expected: FAIL because the watcher does not persist terminal logs, `CancelBuild` still deletes the Job first, and terminal builds do not read from archived files.

- [ ] **Step 3: Implement terminal-state archive wiring**

In `internal/core/build_watcher.go`, call `PersistBuildLogs` on:

- successful Job completion
- failed Job completion

In `internal/services/build.go`, change `CancelBuild` ordering to:

```go
core.GlobalBuildWatcher.StopWatching(buildID)
_ = core.PersistBuildLogs(context.Background(), buildID)
_ = core.CancelBuildJob(...)
// then mark build cancelled and cleanup secrets
```

In `internal/services/build_logs.go`, move `StreamBuildLogs` and branch by status:

```go
switch build.Status {
case entities.BuildStatusPending, entities.BuildStatusCloning, entities.BuildStatusBuilding:
    return streamActiveBuildLogs(c, build)
default:
    return streamPersistedBuildLogs(c, build)
}
```

- [ ] **Step 4: Re-run the focused watcher and service tests**

Run: `go test ./internal/core -run 'TestUpdateBuildFailed_PreservesArchiveFailureMetadata' && go test ./internal/services -run 'Test(StreamBuildLogs|CancelBuild)'`
Expected: PASS

- [ ] **Step 5: Commit the integration slice**

```bash
git add internal/core/build_watcher.go internal/services/build.go internal/services/build_logs.go internal/services/build_logs_test.go internal/core/build_watcher_test.go internal/models/build.go
git commit -m "feat: archive terminal build logs and route reads by build state"
```

### Task 4: Surface archive-state metadata in the build log UI

**Files:**
- Modify: `ui/src/api/builds.ts`
- Modify: `ui/src/components/builds/build-log-viewer.tsx`
- Create: `ui/src/components/builds/build-log-viewer.test.tsx`
- Test: `ui/src/components/builds/build-log-viewer.test.tsx`

- [ ] **Step 1: Write the failing viewer tests**

```tsx
it("shows archive failure messaging for terminal builds")
it("shows archive expired messaging when the archive was deleted")
```

Mock `codeRepositoriesApi.getBuild` and keep the EventSource mocked so the test only checks metadata-driven messaging.

- [ ] **Step 2: Run the focused UI tests to verify they fail**

Run: `npm --prefix ui test -- build-log-viewer.test.tsx`
Expected: FAIL because the `Build` type and viewer do not expose archive status messaging.

- [ ] **Step 3: Implement the minimal UI archive-state messaging**

Extend the shared type:

```ts
log_persist_status?: "pending" | "succeeded" | "failed" | "expired" | ""
log_persist_error?: string
```

Add small terminal-state messaging in `BuildLogViewer`:

- show archive failure details when `log_persist_status === "failed"`
- show an expired hint when `log_persist_status === "expired"`
- keep a generic unavailable hint only for unexpected terminal-state misses

Do not change the existing streaming behavior or layout beyond this messaging.

- [ ] **Step 4: Re-run the focused UI tests**

Run: `npm --prefix ui test -- build-log-viewer.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit the UI slice**

```bash
git add ui/src/api/builds.ts ui/src/components/builds/build-log-viewer.tsx ui/src/components/builds/build-log-viewer.test.tsx
git commit -m "feat: surface build log archive state in viewer"
```

## Chunk 3: Recovery, Cleanup, and Deployment Wiring

### Task 5: Lock archive recovery and cleanup with failing tests

**Files:**
- Create: `internal/core/build_log_maintenance.go`
- Create: `internal/core/build_log_maintenance_test.go`
- Modify: `cmd/api/main.go`
- Test: `internal/core/build_log_maintenance_test.go`

- [ ] **Step 1: Write the failing maintenance tests**

```go
func TestRecoverTerminalBuildLogArchives_RetriesUnpersistedTerminalBuilds(t *testing.T)
func TestDeleteExpiredBuildLogs_RemovesArchiveAndMarksMetadataExpired(t *testing.T)
func TestDeleteExpiredBuildLogs_LeavesActiveArchivesUntouched(t *testing.T)
```

The recovery test should seed:

- one `succeeded` build with `log_persist_status = pending`
- one `failed` build with `log_persist_status = failed`
- one archived-and-expired build with `log_persist_status = expired`
- one active build

Only terminal builds without successful archives and without `expired` status should be retried.

- [ ] **Step 2: Run the focused maintenance tests to verify they fail**

Run: `go test ./internal/core -run 'Test(RecoverTerminalBuildLogArchives|DeleteExpiredBuildLogs)'`
Expected: FAIL because no recovery or cleanup loop exists yet.

- [ ] **Step 3: Implement restart recovery and cleanup helpers**

Add focused functions:

```go
func RecoverTerminalBuildLogArchives(ctx context.Context) error
func DeleteExpiredBuildLogs(ctx context.Context, now time.Time) error
func StartBuildLogMaintenance(ctx context.Context)
```

Wire `cmd/api/main.go` to call:

```go
core.GlobalBuildWatcher.RecoverActiveBuilds()
if err := core.RecoverTerminalBuildLogArchives(context.Background()); err != nil { ... }
go core.StartBuildLogMaintenance(context.Background())
```

Keep cleanup interval as a private daily ticker constant rather than another public config value.

When cleanup deletes an archive successfully, update metadata to:

- `log_persist_status = expired`
- `log_persist_error = ''`
- `log_path = ''`

- [ ] **Step 4: Re-run the focused maintenance tests**

Run: `go test ./internal/core -run 'Test(RecoverTerminalBuildLogArchives|DeleteExpiredBuildLogs)'`
Expected: PASS

- [ ] **Step 5: Commit the maintenance slice**

```bash
git add internal/core/build_log_maintenance.go internal/core/build_log_maintenance_test.go cmd/api/main.go
git commit -m "feat: recover and clean up persisted build logs"
```

### Task 6: Add persistent archive storage wiring to Docker Compose and Helm

**Files:**
- Modify: `deploy/docker/docker-compose.yml`
- Modify: `deploy/helm/ketches/values.yaml`
- Modify: `deploy/helm/ketches/templates/api-deployment.yaml.tpl`
- Create: `deploy/helm/ketches/templates/api-pvc.yaml.tpl`
- Modify: `deploy/helm/ketches/README.md`

- [ ] **Step 1: Render the current deployment configs to capture the missing persistence**

Run: `docker compose -f deploy/docker/docker-compose.yml config`
Expected: PASS, but no API volume mount or build-log env vars are present.

Run: `helm template ketches deploy/helm/ketches`
Expected: PASS, but the API Deployment has no build-log volume mount and no API PVC resource is rendered.

- [ ] **Step 2: Add Docker Compose and Helm persistence wiring**

Docker Compose changes:

- mount a named volume at `/app/data/build-logs`
- pass `BUILD_LOG_BASE_DIR=/app/data/build-logs`
- pass `BUILD_LOG_RETENTION_DAYS=15`

Helm values changes:

```yaml
config:
  buildLogBaseDir: /app/data/build-logs
  buildLogRetentionDays: 15

api:
  persistence:
    enabled: true
    accessModes:
      - ReadWriteOnce
    size: 10Gi
    storageClass: ""
```

Helm template changes:

- mount the archive dir into the API Deployment
- inject the two env vars
- render `api-pvc.yaml.tpl` when persistence is enabled
- document the single-replica assumption in `deploy/helm/ketches/README.md`

- [ ] **Step 3: Re-render deployment configs to verify persistence is present**

Run: `docker compose -f deploy/docker/docker-compose.yml config`
Expected: PASS and the API service includes a build-log volume mount plus archive env vars.

Run: `helm template ketches deploy/helm/ketches`
Expected: PASS and output includes:

- API Deployment volume mount for `/app/data/build-logs`
- API PVC resource when `.Values.api.persistence.enabled` is true

- [ ] **Step 4: Commit the deployment slice**

```bash
git add deploy/docker/docker-compose.yml deploy/helm/ketches/values.yaml deploy/helm/ketches/templates/api-deployment.yaml.tpl deploy/helm/ketches/templates/api-pvc.yaml.tpl deploy/helm/ketches/README.md
git commit -m "feat: add persistent storage for build log archives"
```

## Final Verification

- [ ] **Step 1: Run the targeted backend tests**

Run: `go test ./internal/app ./internal/core ./internal/services`
Expected: PASS

- [ ] **Step 2: Run the targeted frontend test suite**

Run: `npm --prefix ui test -- build-log-viewer.test.tsx`
Expected: PASS

- [ ] **Step 3: Run lightweight deployment config verification**

Run: `docker compose -f deploy/docker/docker-compose.yml config && helm template ketches deploy/helm/ketches >/tmp/ketches-helm-render.yaml`
Expected: PASS

- [ ] **Step 4: Run an end-to-end manual smoke check**

1. Start a build and confirm the viewer still streams live Pod logs.
2. Wait for a `succeeded` build and confirm the archived log remains readable after deleting the build Pod.
3. Trigger a failing build and confirm the archived log shows the failure output.
4. Cancel a running build and confirm partial logs remain readable after cancellation.
5. Reduce retention temporarily and confirm expired archives are deleted by the cleanup loop.

- [ ] **Step 5: Invoke `@superpowers/verification-before-completion` before declaring the implementation done**
