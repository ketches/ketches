package services

import (
	"context"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListClusterCertificates returns paginated certificates scoped to a cluster
func ListClusterCertificates(clusterID string, page, pageSize int, search string) (int64, []entities.Certificate, error) {
	var certs []entities.Certificate
	var total int64
	query := db.DB.Model(&entities.Certificate{}).Where("cluster_id = ? AND scope = ?", clusterID, "cluster")
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at").Find(&certs).Error; err != nil {
		return 0, nil, err
	}
	return total, certs, nil
}

// ListEnvCertificates returns paginated certificates scoped to an environment
func ListEnvCertificates(envID string, page, pageSize int, search string) (int64, []entities.Certificate, error) {
	var certs []entities.Certificate
	var total int64
	query := db.DB.Model(&entities.Certificate{}).Where("env_id = ? AND scope = ?", envID, "env")
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&certs).Error; err != nil {
		return 0, nil, err
	}
	return total, certs, nil
}

// GetCertificate returns a single certificate by ID
func GetCertificate(id string) (*entities.Certificate, error) {
	var cert entities.Certificate
	if err := db.DB.First(&cert, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

// CreateClusterCertificate creates a certificate scoped to a cluster
func CreateClusterCertificate(clusterID string, req *models.CreateCertificateRequest) (*entities.Certificate, error) {
	cert := &entities.Certificate{
		Base: entities.Base{
			ID: uuid.New(),
		},
		Name:        req.Name,
		Description: req.Description,
		Cert:        req.Cert,
		Key:         req.Key,
		Scope:       "cluster",
		ClusterID:   clusterID,
	}

	if err := db.DB.Create(cert).Error; err != nil {
		return nil, err
	}
	return cert, nil
}

// CreateEnvCertificate creates a certificate scoped to an environment
func CreateEnvCertificate(envID string, req *models.CreateCertificateRequest) (*entities.Certificate, error) {
	cert := &entities.Certificate{
		Base: entities.Base{
			ID: uuid.New(),
		},
		Name:        req.Name,
		Description: req.Description,
		Cert:        req.Cert,
		Key:         req.Key,
		Scope:       "env",
		EnvID:       envID,
	}

	if err := db.DB.Create(cert).Error; err != nil {
		return nil, err
	}

	// Sync the env-level Gateway to include the new certificate.
	syncEnvGatewayForCert(cert)

	return cert, nil
}

// UpdateCertificate updates an existing certificate by ID
func UpdateCertificate(id string, req *models.UpdateCertificateRequest) (*entities.Certificate, error) {
	cert, err := GetCertificate(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		cert.Name = *req.Name
	}
	if req.Description != nil {
		cert.Description = *req.Description
	}
	if req.Cert != nil {
		cert.Cert = *req.Cert
	}
	if req.Key != nil {
		cert.Key = *req.Key
	}

	if err := db.DB.Save(cert).Error; err != nil {
		return nil, err
	}

	// Sync the env-level Gateway to reflect certificate changes.
	syncEnvGatewayForCert(cert)

	return cert, nil
}

// DeleteCertificate deletes a certificate by ID
func DeleteCertificate(id string) error {
	cert, err := GetCertificate(id)
	if err != nil {
		return err
	}

	if err := db.DB.Delete(&entities.Certificate{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Sync the env-level Gateway to remove the deleted certificate.
	syncEnvGatewayForCert(cert)
	return nil
}

// syncEnvGatewayForCert re-syncs the env-level Gateway after a certificate
// is created, updated, or deleted. Only env-scoped certificates trigger a sync.
// Errors are intentionally ignored to keep certificate operations non-blocking.
func syncEnvGatewayForCert(cert *entities.Certificate) {
	if cert.Scope != "env" || cert.EnvID == "" {
		return
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", cert.EnvID).Error; err != nil {
		return
	}
	var certs []entities.Certificate
	db.DB.Where("env_id = ? AND scope = ?", cert.EnvID, "env").Find(&certs)
	_ = core.EnsureEnvGateway(context.Background(), &env, certs)
}
