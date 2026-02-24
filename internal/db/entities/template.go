package entities

// Template represents a reusable template resource.
// Fields: Name, Slug, Description, Type, Content, Status, Enabled.
type Template struct {
	Base
	ProjectID string `gorm:"type:varchar(36);index;not null"`

	// Display name
	Name string `gorm:"type:varchar(128);not null"`

	// URL-friendly identifier
	Slug string `gorm:"type:varchar(128);not null;uniqueIndex"`

	// Description of the template
	Description string `gorm:"type:text"`

	// Type categorizes the template (e.g. "application", "service", "job", "cronjob")
	Type string `gorm:"type:varchar(64);not null;default:'application'"`

	// Content holds the template body (YAML, JSON, etc.)
	Content string `gorm:"type:text"`

	// Status: draft, reviewing, published, deprecated
	Status string `gorm:"type:varchar(32);not null;default:'draft'"`

	// Enabled controls whether the template is active
	Enabled bool `gorm:"type:bool;default:true"`

	Project Project `gorm:"foreignKey:ProjectID"`
}

func (Template) TableName() string {
	return "templates"
}
