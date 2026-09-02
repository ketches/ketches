package db

import (
	"sort"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
)

type projectMemberIdentity struct {
	ProjectID string
	UserID    string
}

type projectMemberMigrationRow struct {
	ID          string
	ProjectID   string
	UserID      string
	ProjectRole string
	CreatedAt   time.Time
}

type signupVerificationCodeMigrationRow struct {
	ID        string
	Email     string
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type signupVerificationCodeEmailUpdate struct {
	ID    string
	Email string
}

type envNamespaceIdentity struct {
	ClusterID        string
	ClusterNamespace string
}

type envNamespaceMigrationRow struct {
	ID               string
	ClusterID        string
	ClusterNamespace *string
}

type envNamespaceUpdate struct {
	ID               string
	ClusterNamespace *string
}

type buildRepositoryNumberIdentity struct {
	CodeRepositoryID string
	BuildNumber      int
}

type buildMigrationRow struct {
	ID                      string
	CreatedAt               time.Time
	CodeRepositoryID        *string
	SettingCodeRepositoryID *string
	BuildNumber             int
}

type buildMigrationUpdate struct {
	ID               string
	CodeRepositoryID *string
	BuildNumber      int
	UpdateRepository bool
	UpdateNumber     bool
}

const migrationIDBatchSize = 500

func prepareLegacyUniqueIndexData(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	if err := deduplicateLegacyProjectMembers(database); err != nil {
		return err
	}
	if err := normalizeLegacySignupVerificationCodes(database); err != nil {
		return err
	}
	if err := normalizeLegacyEnvNamespaces(database); err != nil {
		return err
	}
	return backfillLegacyBuildRepositoryNumbers(database)
}

func deduplicateLegacyProjectMembers(database *gorm.DB) error {
	if !database.Migrator().HasTable(&entities.ProjectMember{}) {
		return nil
	}

	var rows []projectMemberMigrationRow
	if err := database.Model(&entities.ProjectMember{}).
		Select("id", "project_id", "user_id", "project_role", "created_at").
		Order("project_id ASC, user_id ASC, id ASC").
		Find(&rows).Error; err != nil {
		return err
	}

	// Legacy databases may contain multiple rows for the same project/user.
	// Keep the strongest role rather than relying on ID ordering: otherwise a
	// duplicate with a lexicographically smaller ID could silently downgrade an
	// owner during migration.
	winners := make(map[projectMemberIdentity]projectMemberMigrationRow, len(rows))
	duplicateIDs := make([]string, 0)
	for _, row := range rows {
		key := projectMemberIdentity{ProjectID: row.ProjectID, UserID: row.UserID}
		winner, exists := winners[key]
		if !exists || projectMemberRowOutranks(row, winner) {
			if exists {
				duplicateIDs = append(duplicateIDs, winner.ID)
			}
			winners[key] = row
			continue
		}
		duplicateIDs = append(duplicateIDs, row.ID)
	}
	if len(duplicateIDs) == 0 {
		return nil
	}

	return deleteLegacyRowsByIDs(database, &entities.ProjectMember{}, duplicateIDs, false)
}

func projectMemberRolePriority(role string) int {
	switch role {
	case "owner":
		return 3
	case "developer":
		return 2
	case "viewer":
		return 1
	default:
		return 0
	}
}

func projectMemberRowOutranks(candidate, current projectMemberMigrationRow) bool {
	candidatePriority := projectMemberRolePriority(candidate.ProjectRole)
	currentPriority := projectMemberRolePriority(current.ProjectRole)
	if candidatePriority != currentPriority {
		return candidatePriority > currentPriority
	}
	if !candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.CreatedAt.Before(current.CreatedAt)
	}
	return candidate.ID < current.ID
}

func deleteLegacyRowsByIDs(database *gorm.DB, model any, ids []string, unscoped bool) error {
	for start := 0; start < len(ids); start += migrationIDBatchSize {
		end := start + migrationIDBatchSize
		if end > len(ids) {
			end = len(ids)
		}

		query := database
		if unscoped {
			query = query.Unscoped()
		}
		if err := query.Where("id IN ?", ids[start:end]).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeLegacySignupVerificationCodes(database *gorm.DB) error {
	if !database.Migrator().HasTable(&entities.SignupVerificationCode{}) {
		return nil
	}

	var rows []signupVerificationCodeMigrationRow
	if err := database.Unscoped().Model(&entities.SignupVerificationCode{}).
		Select("id", "email", "created_at", "deleted_at").
		Order("CASE WHEN deleted_at IS NULL THEN 0 ELSE 1 END ASC").
		Order("created_at DESC").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(rows))
	duplicateIDs := make([]string, 0)
	normalizedEmails := make([]signupVerificationCodeEmailUpdate, 0, len(rows))
	for _, row := range rows {
		normalizedEmail := strings.ToLower(strings.TrimSpace(row.Email))
		if _, exists := seen[normalizedEmail]; exists {
			duplicateIDs = append(duplicateIDs, row.ID)
			continue
		}
		seen[normalizedEmail] = struct{}{}
		if row.Email != normalizedEmail {
			normalizedEmails = append(normalizedEmails, signupVerificationCodeEmailUpdate{
				ID:    row.ID,
				Email: normalizedEmail,
			})
		}
	}
	if len(duplicateIDs) == 0 && len(normalizedEmails) == 0 {
		return nil
	}

	indexName := "idx_signup_verification_codes_email_unique"
	if database.Migrator().HasIndex(&entities.SignupVerificationCode{}, indexName) {
		if err := database.Migrator().DropIndex(&entities.SignupVerificationCode{}, indexName); err != nil {
			return err
		}
	}

	return database.Transaction(func(tx *gorm.DB) error {
		if len(duplicateIDs) > 0 {
			if err := deleteLegacyRowsByIDs(tx, &entities.SignupVerificationCode{}, duplicateIDs, true); err != nil {
				return err
			}
		}
		for _, update := range normalizedEmails {
			if err := tx.Unscoped().Model(&entities.SignupVerificationCode{}).
				Where("id = ?", update.ID).
				UpdateColumn("email", update.Email).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizeLegacyEnvNamespaces(database *gorm.DB) error {
	if !database.Migrator().HasTable(&entities.Env{}) {
		return nil
	}

	var rows []envNamespaceMigrationRow
	if err := database.Unscoped().Model(&entities.Env{}).
		Select("id", "cluster_id", "cluster_namespace").
		Order("CASE WHEN deleted_at IS NULL THEN 0 ELSE 1 END ASC").
		Order("cluster_id ASC").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return err
	}

	seen := make(map[envNamespaceIdentity]struct{}, len(rows))
	updates := make([]envNamespaceUpdate, 0)
	for _, row := range rows {
		if row.ClusterNamespace == nil {
			continue
		}
		normalizedNamespace := strings.TrimSpace(*row.ClusterNamespace)
		if normalizedNamespace == "" {
			updates = append(updates, envNamespaceUpdate{ID: row.ID})
			continue
		}

		key := envNamespaceIdentity{ClusterID: row.ClusterID, ClusterNamespace: normalizedNamespace}
		if _, exists := seen[key]; exists {
			updates = append(updates, envNamespaceUpdate{ID: row.ID})
			continue
		}
		seen[key] = struct{}{}
		if *row.ClusterNamespace != normalizedNamespace {
			namespace := normalizedNamespace
			updates = append(updates, envNamespaceUpdate{ID: row.ID, ClusterNamespace: &namespace})
		}
	}
	if len(updates) == 0 {
		return nil
	}

	indexName := "idx_cluster_namespace"
	if database.Migrator().HasIndex(&entities.Env{}, indexName) {
		if err := database.Migrator().DropIndex(&entities.Env{}, indexName); err != nil {
			return err
		}
	}

	return database.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			var value any
			if update.ClusterNamespace != nil {
				value = *update.ClusterNamespace
			}
			if err := tx.Unscoped().Model(&entities.Env{}).
				Where("id = ?", update.ID).
				UpdateColumn("cluster_namespace", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillLegacyBuildRepositoryNumbers(database *gorm.DB) error {
	if !database.Migrator().HasTable(&entities.Build{}) {
		return nil
	}
	if !database.Migrator().HasColumn(&entities.Build{}, "CodeRepositoryID") {
		if err := database.Migrator().AddColumn(&entities.Build{}, "CodeRepositoryID"); err != nil {
			return err
		}
	}
	if !database.Migrator().HasTable(&entities.BuildSetting{}) ||
		!database.Migrator().HasColumn(&entities.BuildSetting{}, "CodeRepositoryID") {
		return nil
	}

	var rows []buildMigrationRow
	if err := database.Table("builds").
		Select(`builds.id, builds.created_at, builds.code_repository_id,
			builds.build_number, build_settings.code_repository_id AS setting_code_repository_id`).
		Joins("LEFT JOIN build_settings ON build_settings.id = builds.build_setting_id").
		Find(&rows).Error; err != nil {
		return err
	}

	effectiveRepositoryIDs := make(map[string]*string, len(rows))
	maxBuildNumbers := make(map[string]int)
	for _, row := range rows {
		repositoryID := normalizedNullableID(row.CodeRepositoryID)
		if repositoryID == nil {
			repositoryID = normalizedNullableID(row.SettingCodeRepositoryID)
		}
		effectiveRepositoryIDs[row.ID] = repositoryID
		if repositoryID != nil && row.BuildNumber > maxBuildNumbers[*repositoryID] {
			maxBuildNumbers[*repositoryID] = row.BuildNumber
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		leftRepositoryID := effectiveRepositoryIDs[rows[i].ID]
		rightRepositoryID := effectiveRepositoryIDs[rows[j].ID]
		left := ""
		right := ""
		if leftRepositoryID != nil {
			left = *leftRepositoryID
		}
		if rightRepositoryID != nil {
			right = *rightRepositoryID
		}
		if left != right {
			return left < right
		}
		if rows[i].BuildNumber != rows[j].BuildNumber {
			return rows[i].BuildNumber < rows[j].BuildNumber
		}
		if !rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].CreatedAt.Before(rows[j].CreatedAt)
		}
		return rows[i].ID < rows[j].ID
	})

	seen := make(map[buildRepositoryNumberIdentity]struct{}, len(rows))
	updates := make([]buildMigrationUpdate, 0)
	for _, row := range rows {
		repositoryID := effectiveRepositoryIDs[row.ID]
		update := buildMigrationUpdate{ID: row.ID, BuildNumber: row.BuildNumber}
		// Compare the stored value with the effective normalized value, not
		// just their normalized forms. Otherwise a legacy value such as
		// " repo " would remain in the database while the deduplication logic
		// treats it as "repo", and a later unique index could still disagree
		// with the migration's grouping.
		if !nullableStringsEqual(row.CodeRepositoryID, repositoryID) {
			update.CodeRepositoryID = repositoryID
			update.UpdateRepository = true
		}
		if repositoryID != nil {
			key := buildRepositoryNumberIdentity{
				CodeRepositoryID: *repositoryID,
				BuildNumber:      row.BuildNumber,
			}
			if _, exists := seen[key]; exists {
				maxBuildNumbers[*repositoryID]++
				update.BuildNumber = maxBuildNumbers[*repositoryID]
				update.UpdateNumber = true
			} else {
				seen[key] = struct{}{}
			}
		}
		if update.UpdateRepository || update.UpdateNumber {
			updates = append(updates, update)
		}
	}
	if len(updates) == 0 {
		return nil
	}

	indexName := "idx_builds_code_repository_number"
	if database.Migrator().HasIndex(&entities.Build{}, indexName) {
		if err := database.Migrator().DropIndex(&entities.Build{}, indexName); err != nil {
			return err
		}
	}

	return database.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			columns := make(map[string]any, 2)
			if update.UpdateRepository {
				if update.CodeRepositoryID == nil {
					columns["code_repository_id"] = nil
				} else {
					columns["code_repository_id"] = *update.CodeRepositoryID
				}
			}
			if update.UpdateNumber {
				columns["build_number"] = update.BuildNumber
			}
			if err := tx.Model(&entities.Build{}).
				Where("id = ?", update.ID).
				UpdateColumns(columns).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizedNullableID(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func nullableStringsEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
