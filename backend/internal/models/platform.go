package models

type PlatformStatisticsModel struct {
	TotalClusters    int64 `json:"totalClusters"`
	TotalProjects    int64 `json:"totalProjects"`
	TotalUsers       int64 `json:"totalUsers"`
	TotalEnvs        int64 `json:"totalEnvs"`
	TotalApps        int64 `json:"totalApps"`
	TotalAppGateways int64 `json:"totalAppGateways"`
}
