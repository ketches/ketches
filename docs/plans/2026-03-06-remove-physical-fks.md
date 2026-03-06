# Remove Physical Foreign Keys Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove all database-level physical foreign key constraints while maintaining GORM logical associations and ensuring data integrity through the application layer.

**Architecture:** Use GORM's `DisableForeignKeyConstraintWhenMigrating` to prevent new constraint generation. Explicitly add `index` tags to association columns to maintain query performance. Implement application-level cascade logic in the service layer where necessary.

**Tech Stack:** Go, GORM, PostgreSQL, MySQL, SQLite.

---

### Task 1: Update GORM Global Configuration

**Files:**
- Modify: `internal/db/db.go`

**Step 1: Update InitDB to disable foreign key generation**

```go
// internal/db/db.go around line 31
DB, err = gorm.Open(dialector, &gorm.Config{
    DisableForeignKeyConstraintWhenMigrating: true,
})
```

**Step 2: Verify compilation**

Run: `go build ./internal/db/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "chore(db): disable physical foreign key generation in GORM config"
```

---

### Task 2: Explicit Indexing for Association Columns

**Files:**
- Modify: `internal/db/entities/*.go`

**Step 1: Add index tags to all foreign key columns**
Ensure columns like `ProjectID`, `AppID`, `EnvID` in various entities have explicit `index` or `uniqueIndex` tags.

Example for `App` in `internal/db/entities/app.go`:
```go
EnvID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_env_app_slug;index"` // Add explicit index if missing
```

**Step 2: Verify with AutoMigrate**

Run: `make test` (assuming tests trigger AutoMigrate) or check with a local DB.

**Step 3: Commit**

```bash
git add internal/db/entities/*.go
git commit -m "feat(db): add explicit indexes to association columns"
```

---

### Task 3: Migration Logic to Drop Existing Constraints

**Files:**
- Modify: `internal/db/migration.go`

**Step 1: Implement generic constraint dropper for PG/MySQL**

```go
// internal/db/migration.go
func dropAllForeignKeys() error {
    // Implementation to query information_schema and drop constraints
    // This needs to be driver-specific
    return nil 
}
```

**Step 2: Integration into Migrate()**

**Step 3: Commit**

```bash
git add internal/db/migration.go
git commit -m "feat(db): add migration logic to drop existing foreign keys"
```

---

### Task 4: Service Layer Integrity (Cascade Deletion)

**Files:**
- Modify: `internal/services/project.go`
- Modify: `internal/services/app.go`

**Step 1: Ensure DeleteProject explicitly cleans up Envs and Apps in a transaction**

**Step 2: Ensure DeleteApp explicitly cleans up AppEnvVars, AppVolumes, etc.**

**Step 3: Commit**

```bash
git add internal/services/*.go
git commit -m "feat(services): implement application-level cascade deletion"
```
