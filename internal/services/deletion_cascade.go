package services

import (
	"context"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
)

func tableExists(tx *gorm.DB, model any) bool {
	return tx.Migrator().HasTable(model)
}

func validateEnvPermanentDeletionTx(tx *gorm.DB, envID string) error {
	var apps []struct {
		ID        string
		DeletedAt gorm.DeletedAt
	}
	query := tx.Unscoped().Model(&entities.App{}).
		Select("id, deleted_at").
		Where("env_id = ?", envID)
	if err := lockForUpdate(query).Find(&apps).Error; err != nil {
		return err
	}

	activeCount := 0
	for _, application := range apps {
		if !application.DeletedAt.Valid {
			activeCount++
		}
	}
	if activeCount > 0 {
		return app.WrapErrorf(ErrRecycleBinActiveChildren, "environment %s contains %d active applications", envID, activeCount)
	}
	return nil
}

func validateProjectPermanentDeletionTx(tx *gorm.DB, projectID string) error {
	var envs []struct {
		ID        string
		DeletedAt gorm.DeletedAt
	}
	query := tx.Unscoped().Model(&entities.Env{}).
		Select("id, deleted_at").
		Where("project_id = ?", projectID)
	if err := lockForUpdate(query).Find(&envs).Error; err != nil {
		return err
	}

	envIDs := make([]string, 0, len(envs))
	activeEnvCount := 0
	for _, env := range envs {
		envIDs = append(envIDs, env.ID)
		if !env.DeletedAt.Valid {
			activeEnvCount++
		}
	}
	if activeEnvCount > 0 {
		return app.WrapErrorf(ErrRecycleBinActiveChildren, "project %s contains %d active environments", projectID, activeEnvCount)
	}
	if len(envIDs) == 0 {
		return nil
	}

	var apps []struct {
		ID        string
		DeletedAt gorm.DeletedAt
	}
	appQuery := tx.Unscoped().Model(&entities.App{}).
		Select("id, deleted_at").
		Where("env_id IN ?", envIDs)
	if err := lockForUpdate(appQuery).Find(&apps).Error; err != nil {
		return err
	}
	activeAppCount := 0
	for _, application := range apps {
		if !application.DeletedAt.Valid {
			activeAppCount++
		}
	}
	if activeAppCount > 0 {
		return app.WrapErrorf(ErrRecycleBinActiveChildren, "project %s contains %d active applications", projectID, activeAppCount)
	}
	return nil
}

func deleteEnvOwnedRecordsTx(ctx context.Context, tx *gorm.DB, envID string) error {
	var apps []entities.App
	if err := tx.Unscoped().Where("env_id = ?", envID).Find(&apps).Error; err != nil {
		return err
	}
	for i := range apps {
		if err := permanentlyDeleteAppTx(ctx, tx, apps[i].ID, true); err != nil {
			return err
		}
	}

	if tableExists(tx, &entities.AppGroup{}) {
		var groupIDs []string
		if err := tx.Model(&entities.AppGroup{}).Where("env_id = ?", envID).Pluck("id", &groupIDs).Error; err != nil {
			return err
		}
		if len(groupIDs) > 0 && tableExists(tx, &entities.AppGroupMember{}) {
			if err := tx.Unscoped().Where("group_id IN ?", groupIDs).Delete(&entities.AppGroupMember{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.AppGroup{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.AppFavorite{}) {
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.AppFavorite{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.EnvResourceQuota{}) {
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.EnvResourceQuota{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Certificate{}) {
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.Certificate{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Domain{}) {
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.Domain{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Build{}) {
		var buildIDs []string
		if err := tx.Model(&entities.Build{}).Where("build_env_id = ?", envID).Pluck("id", &buildIDs).Error; err != nil {
			return err
		}
		if len(buildIDs) > 0 {
			if tableExists(tx, &entities.BuildDeployment{}) {
				if err := tx.Unscoped().Where("build_id IN ?", buildIDs).Delete(&entities.BuildDeployment{}).Error; err != nil {
					return err
				}
			}
			if tableExists(tx, &entities.DeploymentHistory{}) {
				if err := tx.Unscoped().Model(&entities.DeploymentHistory{}).Where("build_id IN ?", buildIDs).Update("build_id", nil).Error; err != nil {
					return err
				}
			}
			if err := tx.Unscoped().Where("id IN ?", buildIDs).Delete(&entities.Build{}).Error; err != nil {
				return err
			}
		}
	}
	if tableExists(tx, &entities.BuildDeployment{}) {
		if err := tx.Unscoped().Where("env_id = ?", envID).Delete(&entities.BuildDeployment{}).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.OperationLog{}) {
		if err := tx.Unscoped().Model(&entities.OperationLog{}).Where("env_id = ?", envID).Update("env_id", nil).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteUserOwnedRecordsTx(tx *gorm.DB, userID string) error {
	for _, model := range []any{&entities.AppFavorite{}} {
		if !tableExists(tx, model) {
			continue
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(model).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Notification{}) {
		if err := tx.Unscoped().Where("recipient_id = ?", userID).Delete(&entities.Notification{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&entities.Notification{}).Where("sender_id = ?", userID).Update("sender_id", "").Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.OperationLog{}) {
		if err := tx.Unscoped().Model(&entities.OperationLog{}).Where("user_id = ?", userID).Update("user_id", nil).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Build{}) {
		if err := tx.Unscoped().Model(&entities.Build{}).Where("triggered_by = ?", userID).Update("triggered_by", nil).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.Extension{}) {
		if err := tx.Unscoped().Model(&entities.Extension{}).Where("created_by = ?", userID).Update("created_by", nil).Error; err != nil {
			return err
		}
	}
	for _, model := range []any{
		&entities.CollabSprint{},
		&entities.CollabRequirement{},
		&entities.CollabTask{},
		&entities.CollabTestCase{},
		&entities.CollabDefect{},
	} {
		if !tableExists(tx, model) {
			continue
		}
		if err := tx.Unscoped().Model(model).Where("created_by = ?", userID).Update("created_by", "").Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Model(model).Where("updated_by = ?", userID).Update("updated_by", "").Error; err != nil {
			return err
		}
	}
	for _, model := range []any{&entities.CollabRequirement{}, &entities.CollabTask{}, &entities.CollabDefect{}} {
		if !tableExists(tx, model) {
			continue
		}
		if err := tx.Unscoped().Model(model).Where("assignee_id = ?", userID).Update("assignee_id", "").Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteCodeRepositoryProjectRecordsTx(tx *gorm.DB, projectID string) error {
	if !tableExists(tx, &entities.CodeRepository{}) {
		return nil
	}
	var repoIDs []string
	if err := tx.Unscoped().Model(&entities.CodeRepository{}).Where("project_id = ?", projectID).Pluck("id", &repoIDs).Error; err != nil {
		return err
	}
	if len(repoIDs) == 0 {
		return nil
	}
	if err := deleteCodeRepositoryBuildRecordsTx(tx, repoIDs); err != nil {
		return err
	}
	if tableExists(tx, &entities.App{}) {
		if err := tx.Unscoped().Model(&entities.App{}).Where("code_repository_id IN ?", repoIDs).Update("code_repository_id", nil).Error; err != nil {
			return err
		}
	}
	return tx.Unscoped().Where("id IN ?", repoIDs).Delete(&entities.CodeRepository{}).Error
}

func deleteCodeRepositoryBuildRecordsTx(tx *gorm.DB, repoIDs []string) error {
	if len(repoIDs) == 0 {
		return nil
	}

	var buildIDs []string
	if tableExists(tx, &entities.Build{}) {
		query := tx.Table("builds")
		if tableExists(tx, &entities.BuildSetting{}) {
			query = query.Joins("LEFT JOIN build_settings ON build_settings.id = builds.build_setting_id").
				Where("builds.code_repository_id IN ? OR build_settings.code_repository_id IN ?", repoIDs, repoIDs)
		} else {
			query = query.Where("builds.code_repository_id IN ?", repoIDs)
		}
		if err := query.Distinct("builds.id").Pluck("builds.id", &buildIDs).Error; err != nil {
			return err
		}
	}

	if len(buildIDs) > 0 {
		if tableExists(tx, &entities.BuildDeployment{}) {
			if err := tx.Unscoped().Where("build_id IN ?", buildIDs).Delete(&entities.BuildDeployment{}).Error; err != nil {
				return err
			}
		}
		if tableExists(tx, &entities.DeploymentHistory{}) {
			if err := tx.Unscoped().Model(&entities.DeploymentHistory{}).
				Where("build_id IN ?", buildIDs).
				Update("build_id", nil).Error; err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Where("id IN ?", buildIDs).Delete(&entities.Build{}).Error; err != nil {
			return err
		}
	}

	if tableExists(tx, &entities.BuildSetting{}) {
		if err := tx.Unscoped().Where("code_repository_id IN ?", repoIDs).Delete(&entities.BuildSetting{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteProjectOwnedRecordsTx(ctx context.Context, tx *gorm.DB, projectID string) error {
	var envIDs []string
	if tableExists(tx, &entities.Env{}) {
		if err := tx.Unscoped().Model(&entities.Env{}).Where("project_id = ?", projectID).Pluck("id", &envIDs).Error; err != nil {
			return err
		}
	}
	for _, envID := range envIDs {
		if err := deleteEnvOwnedRecordsTx(ctx, tx, envID); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&entities.Env{}, "id = ?", envID).Error; err != nil {
			return err
		}
		if err := deleteRecycleBinDeletionClaim(tx, recycleBinResourceEnvironment, envID); err != nil {
			return err
		}
	}
	if err := deleteCodeRepositoryProjectRecordsTx(tx, projectID); err != nil {
		return err
	}

	models := []any{
		&entities.CollabDefect{},
		&entities.CollabTestRun{},
		&entities.CollabTestCase{},
		&entities.CollabTask{},
		&entities.CollabRequirement{},
		&entities.CollabSprint{},
		&entities.Notification{},
		&entities.Plugin{},
		&entities.ContainerRegistry{},
		&entities.ProjectMember{},
	}
	for _, model := range models {
		if !tableExists(tx, model) {
			continue
		}
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(model).Error; err != nil {
			return err
		}
	}
	if tableExists(tx, &entities.OperationLog{}) {
		if err := tx.Unscoped().Model(&entities.OperationLog{}).Where("project_id = ?", projectID).Update("project_id", nil).Error; err != nil {
			return err
		}
	}
	return nil
}
