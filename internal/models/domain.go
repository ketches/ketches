package models

import "time"

type DomainResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Domain      string    `json:"domain"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
	ClusterID   string    `json:"cluster_id"`
	EnvID       string    `json:"env_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListDomainsResponse struct {
	Items      []DomainResponse   `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type CreateDomainRequest struct {
	Name        string `json:"name" binding:"required"`
	Domain      string `json:"domain" binding:"required"`
	Description string `json:"description"`
}

type UpdateDomainRequest struct {
	Name        *string `json:"name"`
	Domain      *string `json:"domain"`
	Description *string `json:"description"`
}
