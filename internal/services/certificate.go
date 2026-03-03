package services

import (
	"context"
	"errors"

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
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	cert := &entities.Certificate{
		Base: entities.Base{
			ID: uuid.New(),
		},
		Name:        req.Name,
		Description: req.Description,
		Cert:        req.Cert,
		Key:         req.Key,
		Scope:       "cluster",
		ClusterID:   cluster.ID,
	}

	if err := db.DB.Select("id", "name", "description", "cert", "key", "scope", "cluster_id").Create(cert).Error; err != nil {
		return nil, err
	}
	return cert, nil
}

// CreateEnvCertificate creates a certificate scoped to an environment
func CreateEnvCertificate(envID string, req *models.CreateCertificateRequest) (*entities.Certificate, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}
	if env.ClusterID == "" {
		return nil, errors.New("environment is not bound to a cluster")
	}

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
		ClusterID:   env.ClusterID,
	}

	if err := db.DB.Select("id", "name", "description", "cert", "key", "scope", "cluster_id", "env_id").Create(cert).Error; err != nil {
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

	updates := map[string]any{}

	if req.Name != nil {
		cert.Name = *req.Name
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		cert.Description = *req.Description
		updates["description"] = *req.Description
	}
	if req.Cert != nil {
		cert.Cert = *req.Cert
		updates["cert"] = *req.Cert
	}
	if req.Key != nil {
		cert.Key = *req.Key
		updates["key"] = *req.Key
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&entities.Certificate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := db.DB.First(cert, "id = ?", id).Error; err != nil {
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
