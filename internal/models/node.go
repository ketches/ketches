package models

type UpdateNodeLabelsRequest struct {
	Labels map[string]string `json:"labels"`
}

type UpdateNodeAnnotationsRequest struct {
	Annotations map[string]string `json:"annotations"`
}

type UpdateNodeTaintsRequest struct {
	Taints []NodeTaint `json:"taints"`
}

type NodeTaint struct {
	Key    string `json:"taint_key"`
	Value  string `json:"taint_value"`
	Effect string `json:"effect"`
}

type CordonNodeRequest struct {
	Cordon    bool `json:"cordon"`
	EvictPods bool `json:"evict_pods"`
}
