package services

import (
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type recycleBinResourceType string

const (
	recycleBinResourceApp         recycleBinResourceType = "app"
	recycleBinResourceEnvironment recycleBinResourceType = "environment"
	recycleBinResourceProject     recycleBinResourceType = "project"
	recycleBinResourceUser        recycleBinResourceType = "user"
)

type recycleBinDeletionTarget struct {
	resourceType recycleBinResourceType
	resourceID   string
	model        any
	label        string
}

func newRecycleBinDeletionTarget(resourceType recycleBinResourceType, resourceID string, model any, label string) recycleBinDeletionTarget {
	return recycleBinDeletionTarget{
		resourceType: resourceType,
		resourceID:   resourceID,
		model:        model,
		label:        label,
	}
}

func claimRecycleBinDeletionTargets(tx *gorm.DB, targets ...recycleBinDeletionTarget) error {
	for _, target := range targets {
		if err := claimRecycleBinDeletionTarget(tx, target); err != nil {
			return err
		}
	}
	return nil
}

func claimRecycleBinDeletionTarget(tx *gorm.DB, target recycleBinDeletionTarget) error {
	claim := entities.RecycleBinDeletionClaim{
		ID:           uuid.New(),
		ResourceType: string(target.resourceType),
		ResourceID:   target.resourceID,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "resource_type"},
			{Name: "resource_id"},
		},
		DoNothing: true,
	}).Create(&claim).Error; err != nil {
		return err
	}

	// This conditional write serializes the claim with a concurrent restore.
	// The claim is rolled back when the resource was restored first.
	result := tx.Unscoped().Model(target.model).
		Where("id = ? AND deleted_at IS NOT NULL", target.resourceID).
		UpdateColumn("updated_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	// Some MySQL configurations report only changed rows. Distinguish that
	// case from a resource that no longer satisfies the soft-delete predicate.
	var softDeletedCount int64
	if err := tx.Unscoped().Model(target.model).
		Where("id = ? AND deleted_at IS NOT NULL", target.resourceID).
		Count(&softDeletedCount).Error; err != nil {
		return err
	}
	if softDeletedCount > 0 {
		return nil
	}

	var resourceCount int64
	if err := tx.Unscoped().Model(target.model).
		Where("id = ?", target.resourceID).
		Count(&resourceCount).Error; err != nil {
		return err
	}
	if resourceCount == 0 {
		return app.WrapErrorf(ErrRecycleBinResourceNotFound, "%s %s", target.label, target.resourceID)
	}
	return app.WrapErrorf(ErrRecycleBinResourceActive, "%s %s", target.label, target.resourceID)
}

func deleteRecycleBinDeletionClaim(tx *gorm.DB, resourceType recycleBinResourceType, resourceID string) error {
	return tx.Where("resource_type = ? AND resource_id = ?", string(resourceType), resourceID).
		Delete(&entities.RecycleBinDeletionClaim{}).Error
}

func hasRecycleBinDeletionClaim(tx *gorm.DB, resourceType recycleBinResourceType, resourceID string) (bool, error) {
	var count int64
	if err := tx.Model(&entities.RecycleBinDeletionClaim{}).
		Where("resource_type = ? AND resource_id = ?", string(resourceType), resourceID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func restoreAppTx(tx *gorm.DB, appID string) error {
	result := tx.Unscoped().Model(&entities.App{}).
		Where("apps.id = ? AND apps.deleted_at IS NOT NULL", appID).
		Where(`EXISTS (
			SELECT 1
			FROM envs
			JOIN projects ON projects.id = envs.project_id
			WHERE envs.id = apps.env_id
			  AND envs.deleted_at IS NULL
			  AND projects.deleted_at IS NULL
		)`).
		Where(`NOT EXISTS (
			SELECT 1
			FROM recycle_bin_deletion_claims AS claims
			WHERE (claims.resource_type = ? AND claims.resource_id = apps.id)
			   OR (claims.resource_type = ? AND claims.resource_id = apps.env_id)
			   OR (claims.resource_type = ? AND claims.resource_id = (
				   SELECT envs.project_id FROM envs WHERE envs.id = apps.env_id
			   ))
		)`, recycleBinResourceApp, recycleBinResourceEnvironment, recycleBinResourceProject).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return classifyAppRestoreFailure(tx, appID)
}

func classifyAppRestoreFailure(tx *gorm.DB, appID string) error {
	var row struct {
		DeletedAt        gorm.DeletedAt
		EnvID            string
		ParentEnvID      string
		EnvDeletedAt     gorm.DeletedAt
		ProjectID        string
		ParentProjectID  string
		ProjectDeletedAt gorm.DeletedAt
	}
	if err := tx.Unscoped().Table("apps").
		Select(`apps.deleted_at, apps.env_id, envs.id AS parent_env_id,
			envs.deleted_at AS env_deleted_at, envs.project_id,
			projects.id AS parent_project_id, projects.deleted_at AS project_deleted_at`).
		Joins("LEFT JOIN envs ON envs.id = apps.env_id").
		Joins("LEFT JOIN projects ON projects.id = envs.project_id").
		Where("apps.id = ?", appID).
		Take(&row).Error; err != nil {
		return recycleBinActionError(appID, err)
	}
	if !row.DeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinResourceActive, "app %s", appID)
	}
	claimed, err := anyRecycleBinDeletionClaim(tx,
		claimKey{recycleBinResourceApp, appID},
		claimKey{recycleBinResourceEnvironment, row.EnvID},
		claimKey{recycleBinResourceProject, row.ProjectID},
	)
	if err != nil {
		return err
	}
	if claimed {
		return app.WrapErrorf(ErrRecycleBinResourceDeleting, "app %s", appID)
	}
	if row.ParentEnvID == "" || row.ParentProjectID == "" {
		return app.WrapErrorf(ErrRecycleBinResourceNotFound, "app %s parent resource is missing", appID)
	}
	if row.EnvDeletedAt.Valid || row.ProjectDeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinParentDeleted, "app %s", appID)
	}
	return app.WrapErrorf(ErrRecycleBinResourceNotFound, "app %s could not be restored", appID)
}

func restoreEnvTx(tx *gorm.DB, envID string) error {
	result := tx.Unscoped().Model(&entities.Env{}).
		Where("envs.id = ? AND envs.deleted_at IS NOT NULL", envID).
		Where(`EXISTS (
			SELECT 1
			FROM projects
			WHERE projects.id = envs.project_id
			  AND projects.deleted_at IS NULL
		)`).
		Where(`NOT EXISTS (
			SELECT 1
			FROM recycle_bin_deletion_claims AS claims
			WHERE (claims.resource_type = ? AND claims.resource_id = envs.id)
			   OR (claims.resource_type = ? AND claims.resource_id = envs.project_id)
		)`, recycleBinResourceEnvironment, recycleBinResourceProject).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return classifyEnvRestoreFailure(tx, envID)
}

func classifyEnvRestoreFailure(tx *gorm.DB, envID string) error {
	var row struct {
		DeletedAt        gorm.DeletedAt
		ProjectID        string
		ParentProjectID  string
		ProjectDeletedAt gorm.DeletedAt
	}
	if err := tx.Unscoped().Table("envs").
		Select(`envs.deleted_at, envs.project_id, projects.id AS parent_project_id,
			projects.deleted_at AS project_deleted_at`).
		Joins("LEFT JOIN projects ON projects.id = envs.project_id").
		Where("envs.id = ?", envID).
		Take(&row).Error; err != nil {
		return recycleBinActionError(envID, err)
	}
	if !row.DeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinResourceActive, "environment %s", envID)
	}
	claimed, err := anyRecycleBinDeletionClaim(tx,
		claimKey{recycleBinResourceEnvironment, envID},
		claimKey{recycleBinResourceProject, row.ProjectID},
	)
	if err != nil {
		return err
	}
	if claimed {
		return app.WrapErrorf(ErrRecycleBinResourceDeleting, "environment %s", envID)
	}
	if row.ParentProjectID == "" {
		return app.WrapErrorf(ErrRecycleBinResourceNotFound, "environment %s parent project is missing", envID)
	}
	if row.ProjectDeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinParentDeleted, "environment %s", envID)
	}
	return app.WrapErrorf(ErrRecycleBinResourceNotFound, "environment %s could not be restored", envID)
}

func restoreProjectTx(tx *gorm.DB, projectID string) error {
	result := tx.Unscoped().Model(&entities.Project{}).
		Where("projects.id = ? AND projects.deleted_at IS NOT NULL", projectID).
		Where(`NOT EXISTS (
			SELECT 1
			FROM recycle_bin_deletion_claims AS claims
			WHERE claims.resource_type = ? AND claims.resource_id = projects.id
		)`, recycleBinResourceProject).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return classifySimpleRestoreFailure(tx, &entities.Project{}, recycleBinResourceProject, projectID, "project")
}

func restoreUserTx(tx *gorm.DB, userID string) error {
	result := tx.Unscoped().Model(&entities.User{}).
		Where("users.id = ? AND users.deleted_at IS NOT NULL", userID).
		Where(`NOT EXISTS (
			SELECT 1
			FROM recycle_bin_deletion_claims AS claims
			WHERE claims.resource_type = ? AND claims.resource_id = users.id
		)`, recycleBinResourceUser).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return classifySimpleRestoreFailure(tx, &entities.User{}, recycleBinResourceUser, userID, "user")
	}
	return restoreOwnedProjects(tx, userID)
}

func classifySimpleRestoreFailure(tx *gorm.DB, model any, resourceType recycleBinResourceType, resourceID, label string) error {
	var count int64
	if err := tx.Unscoped().Model(model).Where("id = ?", resourceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return app.WrapErrorf(ErrRecycleBinResourceNotFound, "%s %s", label, resourceID)
	}
	claimed, err := hasRecycleBinDeletionClaim(tx, resourceType, resourceID)
	if err != nil {
		return err
	}
	if claimed {
		return app.WrapErrorf(ErrRecycleBinResourceDeleting, "%s %s", label, resourceID)
	}
	return app.WrapErrorf(ErrRecycleBinResourceActive, "%s %s", label, resourceID)
}

type claimKey struct {
	resourceType recycleBinResourceType
	resourceID   string
}

func anyRecycleBinDeletionClaim(tx *gorm.DB, keys ...claimKey) (bool, error) {
	for _, key := range keys {
		if key.resourceID == "" {
			continue
		}
		claimed, err := hasRecycleBinDeletionClaim(tx, key.resourceType, key.resourceID)
		if err != nil {
			return false, err
		}
		if claimed {
			return true, nil
		}
	}
	return false, nil
}
