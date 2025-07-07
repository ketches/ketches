package models

type AppSchedulingRuleTolerationModel struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
}

type AppSchedulingRuleModel struct {
	RuleID       string                             `json:"ruleID"`
	AppID        string                             `json:"appID"`
	RuleType     string                             `json:"ruleType"`
	NodeName     string                             `json:"nodeName,omitempty"`
	NodeSelector []string                           `json:"nodeSelector,omitempty"`
	NodeAffinity []string                           `json:"nodeAffinity,omitempty"`
	Tolerations  []AppSchedulingRuleTolerationModel `json:"tolerations,omitempty"`
}

type SetAppSchedulingRuleRequest struct {
	AppID        string                             `json:"-" uri:"appID"`
	RuleType     string                             `json:"ruleType"`
	NodeName     string                             `json:"nodeName,omitempty"`
	NodeSelector []string                           `json:"nodeSelector,omitempty"`
	NodeAffinity []string                           `json:"nodeAffinity,omitempty"`
	Tolerations  []AppSchedulingRuleTolerationModel `json:"tolerations,omitempty"`
}

type SetAppSchedulingRuleTolerationRequest struct {
	AppID       string                             `json:"-" uri:"appID"`
	RuleID      string                             `json:"-" uri:"ruleID"`
	Tolerations []AppSchedulingRuleTolerationModel `json:"tolerations"`
}

type GetAppSchedulingRuleRequest struct {
	AppID string `uri:"appID" binding:"required"`
}
