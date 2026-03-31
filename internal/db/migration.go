package db

import (
	"log"

	"github.com/ketches/ketches/internal/db/entities"
)

func Migrate() error {
	log.Printf("Running AutoMigrate...")

	if err := DB.AutoMigrate(
		&entities.User{},
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
		&entities.UserAIProvider{},
		&entities.ProjectAIProvider{},
		&entities.BuilderSession{},
		&entities.BuilderMessage{},
		&entities.BuilderRun{},
		&entities.BuilderExecutorHandle{},
		&entities.BuilderRunEvent{},
		&entities.BuilderWorkspace{},
		&entities.BuilderArtifact{},
		&entities.BuilderExport{},
		&entities.BuilderOutputSnapshot{},
		&entities.BuilderOutputSnapshotFile{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.CodeRepository{},
		&entities.DeploymentHistory{},
		&entities.Certificate{},
		&entities.Extension{},
		&entities.ClusterExtension{},
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

	return nil
}
