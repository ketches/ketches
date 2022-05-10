package app

type CreateAppReq struct {
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Namespace string `json:"namespace"`
	Status    uint8  `json:"status"`
	ClusterID string `json:"cluster_id"`
	TenantID  string `json:"tenant_id"`
}

type CreateAppResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Namespace string `json:"namespace"`
	Status    uint8  `json:"status"`
	ClusterID string `json:"cluster_id"`
	TenantID  string `json:"tenant_id"`
}
