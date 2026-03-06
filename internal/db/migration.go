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
		&entities.AppGroup{},
		&entities.AppGroupMember{},
		&entities.AppFavorite{},
		&entities.ContainerRegistry{},
		&entities.AppBuildConfig{},
		&entities.Build{},
		&entities.CodeRepository{},
		&entities.CodeRepositoryBuildConfig{},
		&entities.DeploymentHistory{},
		&entities.Certificate{},
		&entities.Extension{},
		&entities.ClusterExtension{},
	); err != nil {
		return err
	}
	// No physical foreign keys will be created due to DisableForeignKeyConstraintWhenMigrating: true
	// existing physical foreign key cleanup logic for legacy DBs is not needed for a clean start
	return nil
}
