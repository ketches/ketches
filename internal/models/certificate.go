package models

import "time"

// CertificateResponse represents the API response for a certificate.
// Cert and Key fields are intentionally excluded for security.
type CertificateResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Scope         string    `json:"scope"`
	ClusterID     string    `json:"cluster_id"`
	EnvID         string    `json:"env_id,omitempty"`
	HasPrivateKey bool      `json:"has_private_key"`
	CreatedAt     time.Time `json:"created_at"`
}

// ListCertificatesResponse represents the paginated list of certificates
type ListCertificatesResponse struct {
	Items      []CertificateResponse `json:"items"`
	Pagination PaginationResponse    `json:"pagination"`
}

// CreateCertificateRequest represents the request to create a certificate
type CreateCertificateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Cert        string `json:"cert" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Scope       string `json:"scope" binding:"required,oneof=cluster env"`
}

// UpdateCertificateRequest represents the request to update a certificate.
// All fields are optional (pointer types).
type UpdateCertificateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Cert        *string `json:"cert"`
	Key         *string `json:"key"`
}
