package entities

type Audit struct {
	UUIDBase
	SourceKey     string `json:"sourceKey" gorm:"not null"`     // Unique identifier for the source of the audit event
	SourceValue   string `json:"sourceValue"`                   // Value associated with the source key, can be empty if not applicable
	RequestMethod string `json:"requestMethod" gorm:"not null"` // HTTP method of the request (e.g., GET, POST, PUT, DELETE)
	RequestPath   string `json:"requestPath"`                   // Path of the request, can be empty if not applicable
	AuditBase
}
