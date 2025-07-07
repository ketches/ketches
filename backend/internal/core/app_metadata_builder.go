package core

import (
	"context"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
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

	appGateways, err := orm.AllAppGateways(b.appEntity.ID)
	if err != nil {
		return nil, err
	}

	appProbes, err := orm.AllAppProbes(b.appEntity.ID)
	if err != nil {
		return nil, err
	}

	result := &AppMetadata{
		AppID:            b.appEntity.ID,
		AppSlug:          b.appEntity.Slug,
		DisplayName:      b.appEntity.DisplayName,
		Description:      b.appEntity.Description,
		AppType:          b.appEntity.AppType,
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
		EnvID:            b.appEntity.EnvID,
		EnvSlug:          b.appEntity.EnvSlug,
		ProjectID:        b.appEntity.ProjectID,
		ProjectSlug:      b.appEntity.ProjectSlug,
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

	for _, gateway := range appGateways {
		result.Gateways = append(result.Gateways, AppMetadataGateway{
			Port:        gateway.Port,
			Protocol:    gateway.Protocol,
			Exposed:     gateway.Exposed,
			Domain:      gateway.Domain,
			Path:        gateway.Path,
			GatewayPort: gateway.GatewayPort,
		})
	}

	for _, probe := range appProbes {
		result.Probes = append(result.Probes, AppMetadataProbe{
			Type:                probe.Type,
			InitialDelaySeconds: probe.InitialDelaySeconds,
			PeriodSeconds:       probe.PeriodSeconds,
			TimeoutSeconds:      probe.TimeoutSeconds,
			SuccessThreshold:    probe.SuccessThreshold,
			FailureThreshold:    probe.FailureThreshold,
			ProbeMode:           probe.ProbeMode,
			HTTPGetPath:         probe.HTTPGetPath,
			HTTPGetPort:         probe.HTTPGetPort,
			TCPSocketPort:       probe.TCPSocketPort,
			ExecCommand:         probe.ExecCommand,
		})
	}

	appSchedulingRule, err := orm.GetAppSchedulingRule(b.ctx, b.appEntity.ID)
	if err != nil {
		return nil, err
	}
	if appSchedulingRule != nil {
		schedulingRule := &AppMetadataSchedulingRule{
			RuleType:     appSchedulingRule.RuleType,
			NodeName:     appSchedulingRule.NodeName,
			NodeSelector: strings.Split(appSchedulingRule.NodeSelector, ","),
			NodeAffinity: strings.Split(appSchedulingRule.NodeAffinity, ","),
			Tolerations:  make([]Toleration, 0, len(appSchedulingRule.Tolerations)),
		}
		if appSchedulingRule.Tolerations != "" {
			if err := json.Unmarshal([]byte(appSchedulingRule.Tolerations), &schedulingRule.Tolerations); err != nil {
				return nil, app.NewError(http.StatusInternalServerError, "无法解析调度规则容忍设置")
			}
		}
		result.SchedulingRule = schedulingRule
	}

	return result, nil
}

func splitAccessModes(modes string) []string {
	if modes == "" {
		return nil
	}
	return strings.Split(modes, ";")
}
