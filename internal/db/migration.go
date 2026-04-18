package db

import (
	"log/slog"

	"github.com/ketches/ketches/internal/db/entities"
)

func Migrate() error {
	slog.Info("running database automigrate")

	if err := migrateClusterGatewayAddressToGatewayHost(); err != nil {
		return err
	}

	if err := DB.AutoMigrate(
		&entities.User{},
		&entities.SignupVerificationCode{},
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

	if err := migrateClusterGatewayProviderUniqueIndex(); err != nil {
		return err
	}

	return nil
}

func migrateClusterGatewayAddressToGatewayHost() error {
	if DB == nil {
		return nil
	}

	if !DB.Migrator().HasTable(&entities.Cluster{}) {
		return nil
	}

	hasOldColumn := DB.Migrator().HasColumn("clusters", "gateway_address")
	hasNewColumn := DB.Migrator().HasColumn("clusters", "gateway_host")

	if !hasOldColumn || hasNewColumn {
		return nil
	}

	return DB.Migrator().RenameColumn("clusters", "gateway_address", "gateway_host")
}

func migrateClusterGatewayProviderUniqueIndex() error {
	if DB == nil {
		return nil
	}

	if !DB.Migrator().HasTable(&entities.ClusterGatewayProvider{}) {
		return nil
	}

	const indexName = "uidx_cluster_gateway_class"
	if DB.Migrator().HasIndex(&entities.ClusterGatewayProvider{}, indexName) {
		if err := DB.Migrator().DropIndex(&entities.ClusterGatewayProvider{}, indexName); err != nil {
			return err
		}
	}

	return DB.Migrator().CreateIndex(&entities.ClusterGatewayProvider{}, indexName)
}
