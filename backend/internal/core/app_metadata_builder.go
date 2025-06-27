package core

import (
	"context"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
)

type AppMetadataBuilder interface {
	Build() (*AppMetadata, app.Error)
}

type appMetadataBuilder struct {
	ctx       context.Context
	appEntity *entities.App
}

func NewAppMetadataBuilder(ctx context.Context, appID string) (AppMetadataBuilder, app.Error) {
	appEntity, err := orm.GetAppByID(context.Background(), appID) // Ensure the app exists
	if err != nil {
		return nil, err
	}

	return &appMetadataBuilder{
		ctx:       ctx,
		appEntity: appEntity,
	}, nil
}

func NewAppMetadataBuilderFromAppEntity(ctx context.Context, appEntity *entities.App) AppMetadataBuilder {
	return &appMetadataBuilder{
		ctx:       ctx,
		appEntity: appEntity,
	}
}

func (b *appMetadataBuilder) Build() (*AppMetadata, app.Error) {
	appEnvVars, err := orm.AllAppEnvVars(b.appEntity.ID)
	if err != nil {
		return nil, err
	}

	appVolumes, err := orm.AllAppVolumes(b.appEntity.ID)
	if err != nil {
		return nil, err
	}

	appPorts, err := orm.AllAppPorts(b.appEntity.ID)
	if err != nil {
		return nil, err
	}

	result := &AppMetadata{
		Slug:             b.appEntity.Slug,
		DisplayName:      b.appEntity.DisplayName,
		Description:      b.appEntity.Description,
		WorkloadType:     b.appEntity.WorkloadType,
		RequestCPU:       b.appEntity.RequestCPU,
		RequestMemory:    b.appEntity.RequestMemory,
		LimitCPU:         b.appEntity.LimitCPU,
		LimitMemory:      b.appEntity.LimitMemory,
		Replicas:         b.appEntity.Replicas,
		ContainerImage:   b.appEntity.ContainerImage,
		RegistryUsername: b.appEntity.RegistryUsername,
		RegistryPassword: b.appEntity.RegistryPassword,
		ContainerCommand: b.appEntity.ContainerCommand,
		Edition:          b.appEntity.Edition,
		ClusterNamespace: b.appEntity.ClusterNamespace,
	}

	for _, envVar := range appEnvVars {
		result.EnvVars = append(result.EnvVars, AppMetadataEnvVar{
			Key:   envVar.Key,
			Value: envVar.Value,
		})
	}

	for _, volume := range appVolumes {
		result.Volumes = append(result.Volumes, AppMetadataVolume{
			Slug:         volume.Slug,
			MountPath:    volume.MountPath,
			SubPath:      volume.SubPath,
			StorageClass: volume.StorageClass,
			AccessModes:  splitAccessModes(volume.AccessModes),
			VolumeType:   volume.VolumeType,
			Capacity:     volume.Capacity,
			VolumeMode:   volume.VolumeMode,
		})
	}

	for _, port := range appPorts {
		result.Ports = append(result.Ports, AppMetadataPort{
			Port:     port.Port,
			Protocol: port.Protocol,
		})
	}
	return result, nil
}

func splitAccessModes(modes string) []string {
	if modes == "" {
		return nil
	}
	return strings.Split(modes, ";")
}
