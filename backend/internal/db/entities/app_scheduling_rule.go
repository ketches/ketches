package entities

type AppSchedulingRule struct {
	UUIDBase
	AppID        string `json:"appID" gorm:"not null;uniqueIndex;size:36"`
	RuleType     string `json:"ruleType" gorm:"not null;size:32"`
	NodeName     string `json:"nodeName" gorm:"size:255"`
	NodeSelector string `json:"nodeSelector" gorm:"type:text"`
	NodeAffinity string `json:"nodeAffinity" gorm:"type:text"`
	Tolerations  string `json:"tolerations" gorm:"type:text"`
	AuditBase
}
