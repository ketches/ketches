package gateway

type CreateGatewayRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	LastOperatorId uint64 `json:"lastOperatorId"`
	ClusterId      string `json:"clusterId"`
}

type CreateGatewayResponse struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
