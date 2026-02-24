package models

import "time"

type ClusterIntegrationResponse struct {
	ID              string    `json:"id"`
	ClusterID       string    `json:"cluster_id"`
	IntegrationType string    `json:"integration_type"`
	Name            string    `json:"name"`
	Endpoint        string    `json:"endpoint"`
	Namespace       string    `json:"namespace,omitempty"`
	ServiceName     string    `json:"service_name,omitempty"`
	ServicePort     int       `json:"service_port,omitempty"`
	Username        string    `json:"username,omitempty"`
	SkipTLSVerify   bool      `json:"skip_tls_verify"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateClusterIntegrationRequest struct {
	IntegrationType string `json:"integration_type" binding:"required,oneof=prometheus grafana loki alertmanager"`
	Name            string `json:"name" binding:"required"`
	Endpoint        string `json:"endpoint" binding:"required_without=ServiceName"`
	Namespace       string `json:"namespace"`
	ServiceName     string `json:"service_name"`
	ServicePort     int    `json:"service_port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Token           string `json:"token"`
	CACert          string `json:"ca_cert"`
	SkipTLSVerify   bool   `json:"skip_tls_verify"`
	Enabled         bool   `json:"enabled"`
}

type UpdateClusterIntegrationRequest struct {
	Name          *string `json:"name"`
	Endpoint      *string `json:"endpoint"`
	Namespace     *string `json:"namespace"`
	ServiceName   *string `json:"service_name"`
	ServicePort   *int    `json:"service_port"`
	Username      *string `json:"username"`
	Password      *string `json:"password"`
	Token         *string `json:"token"`
	CACert        *string `json:"ca_cert"`
	SkipTLSVerify *bool   `json:"skip_tls_verify"`
	Enabled       *bool   `json:"enabled"`
}
