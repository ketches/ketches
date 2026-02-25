package db

import (
	"log"

	"github.com/ketches/ketches/internal/db/entities"
)

func Migrate() error {
	log.Printf("Running AutoMigrate...")

	if DB.Dialector.Name() == "mysql" {
		DB.Exec("SET FOREIGN_KEY_CHECKS = 0;")
		defer DB.Exec("SET FOREIGN_KEY_CHECKS = 1;")
	}

	if err := DB.AutoMigrate(
		&entities.User{},
		&entities.Cluster{},
		&entities.ClusterIntegration{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppGateway{},
		&entities.AppProbe{},
		&entities.AppConfigFile{},
		&entities.AppSchedulingRule{},
		&entities.AppAutoScaling{},
		&entities.Plugin{},
		&entities.AppPlugin{},
		&entities.ContainerRegistry{},
		&entities.AppBuildConfig{},
		&entities.Build{},
		&entities.CodeRepository{},
		&entities.CodeRepositoryBuildConfig{},
		&entities.DeploymentHistory{},
		&entities.Certificate{},
		&entities.ExtensionCatalogItem{},
	); err != nil {
		return err
	}
	if DB.Dialector.Name() == "mysql" {
		for _, fk := range []string{"fk_apps_builds", "fk_builds_app_id"} {
			_ = DB.Exec("ALTER TABLE builds DROP FOREIGN KEY " + fk).Error
		}
		for _, fk := range []string{"fk_builds_build_config", "fk_builds_build_config_id"} {
			_ = DB.Exec("ALTER TABLE builds DROP FOREIGN KEY " + fk).Error
		}
		_ = DB.Exec("ALTER TABLE builds MODIFY COLUMN app_id VARCHAR(36) NULL").Error
		_ = DB.Exec("ALTER TABLE builds MODIFY COLUMN build_config_id VARCHAR(36) NULL").Error
	}
	return nil
}
