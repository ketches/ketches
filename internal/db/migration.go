package db

import (
	"log/slog"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
)

// Migrate keeps the development database schema aligned with the entity definitions.
func Migrate() error {
	slog.Info("running database automigrate")
	hadUserTable := DB.Migrator().HasTable(&entities.User{})
	hadPasswordChangeColumn := DB.Migrator().HasColumn(&entities.User{}, "MustChangePassword")

	if err := prepareLegacyUniqueIndexData(DB); err != nil {
		return err
	}

	if err := DB.AutoMigrate(
		&entities.User{},
		&entities.RecycleBinDeletionClaim{},
		&entities.SignupVerificationCode{},
		&entities.AuthRateLimit{},
		&entities.Cluster{},
		&entities.ClusterIntegration{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.EnvResourceQuota{},
		&entities.App{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
		&entities.AppProbe{},
		&entities.AppConfigFile{},
		&entities.AppSchedulingRule{},
		&entities.AppAutoScaling{},
		&entities.Plugin{},
		&entities.AppPlugin{},
		&entities.AppGroup{},
		&entities.AppGroupMember{},
		&entities.AppFavorite{},
		&entities.ContainerRegistry{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.CodeRepository{},
		&entities.DeploymentHistory{},
		&entities.Certificate{},
		&entities.Domain{},
		&entities.Extension{},
		&entities.ClusterExtension{},
		&entities.ClusterGatewayProvider{},
		&entities.OperationLog{},
		&entities.SystemSetting{},
		&entities.CollabSprint{},
		&entities.CollabRequirement{},
		&entities.CollabTask{},
		&entities.CollabTestCase{},
		&entities.CollabTestRun{},
		&entities.CollabDefect{},
		&entities.Notification{},
	); err != nil {
		return err
	}

	if hadUserTable && !hadPasswordChangeColumn {
		return requireLegacyBootstrapAdminPasswordChange(DB)
	}
	return nil
}

func requireLegacyBootstrapAdminPasswordChange(database *gorm.DB) error {
	return database.Model(&entities.User{}).
		Where("role = ? AND email LIKE ?", app.UserRoleAdmin, "%@local.ketches").
		Update("must_change_password", true).Error
}
