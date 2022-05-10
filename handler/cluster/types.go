package cluster

type GetClusterRequest struct {
	ClusterId string `json:"clusterId"`
}

type SingleClusterRequest struct {
	Id             string           `json:"id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	KubeConfig     string           `json:"kubeConfig"`
	Formatting     enums.Formatting `json:"formatting"`
	LastOperatorId uint64           `json:"lastOperatorId"`
}

type SingleClusterResponse struct {
	Id           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	KubeConfig   string           `json:"kubeConfig"`
	Formatting   enums.Formatting `json:"formatting"`
	LastOperator string           `json:"lastOperator"`
}

type GetClusterResponse struct {
	SingleClusterResponse
}

type CreateClusterRequest struct {
	Name           string                      `json:"name"`
	Description    string                      `json:"description"`
	KubeConfig     string                      `json:"kubeConfig"`
	Formatting     enums.Formatting            `json:"formatting"`
	ImportMode     enums.KubeClusterImportMode `json:"importMode"`
	Certificate    string                      `json:"certificate"`
	Server         string                      `json:"Server"`
	Token          string                      `json:"token"`
	LastOperatorId uint64                      `json:"lastOperatorId"`
}

type CreateClusterResponse struct {
	SingleClusterResponse
}

type UpdateClusterRequest struct {
	SingleClusterRequest
}

type UpdateClusterResponse struct {
	SingleClusterResponse
}

type PagingClusterRequest struct {
	models.PagingRequest
	Filter string `json:"filter"`
}

type PagingClusterResponse struct {
	Total    int64
	Clusters []SingleClusterResponse
}
