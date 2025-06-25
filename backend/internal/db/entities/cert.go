package entities

type Cert struct {
	UUIDBase
	Slug    string `json:"slug" gorm:"not null;uniqueIndex;size:36"` // Certificate slug, typically a URL-friendly name
	Level   string `json:"level" gorm:"not null;size:16"`            // Certificate level, e.g., 'platform', 'project'
	Domain  string `json:"domain" gorm:"not null;size:255"`          // Domain name for the certificate
	TLSCert string `json:"tlsCert" gorm:"not null;type:text"`        // TLS certificate in PEM format
	TLSKey  string `json:"tlsKey" gorm:"not null;type:text"`         // TLS private key for the certificate
	AuditBase
}
