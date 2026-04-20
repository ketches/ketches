package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListClusterCertificates returns paginated certificates for a cluster
func ListClusterCertificates(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, certs, err := services.ListClusterCertificates(clusterID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.CertificateResponse, 0, len(certs))
	for _, cert := range certs {
		res = append(res, models.CertificateResponse{
			ID:          cert.ID,
			Name:        cert.Name,
			Description: cert.Description,
			Scope:       cert.Scope,
			ClusterID:   cert.ClusterID,
			EnvID:       derefCertString(cert.EnvID),
			CreatedAt:   cert.CreatedAt,
		})
	}

	api.Success(c, models.ListCertificatesResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

// ListEnvCertificates returns paginated certificates for an environment
func ListEnvCertificates(c *gin.Context) {
	envID := c.Param("envID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, certs, err := services.ListEnvCertificates(envID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.CertificateResponse, 0, len(certs))
	for _, cert := range certs {
		res = append(res, models.CertificateResponse{
			ID:          cert.ID,
			Name:        cert.Name,
			Description: cert.Description,
			Scope:       cert.Scope,
			ClusterID:   cert.ClusterID,
			EnvID:       derefCertString(cert.EnvID),
			CreatedAt:   cert.CreatedAt,
		})
	}

	api.Success(c, models.ListCertificatesResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

// GetCertificate returns a single certificate by ID
func GetCertificate(c *gin.Context) {
	certID := c.Param("certID")

	cert, err := services.GetCertificate(certID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, models.CertificateResponse{
		ID:          cert.ID,
		Name:        cert.Name,
		Description: cert.Description,
		Scope:       cert.Scope,
		ClusterID:   cert.ClusterID,
		EnvID:       derefCertString(cert.EnvID),
		CreatedAt:   cert.CreatedAt,
	})
}

// CreateClusterCertificate creates a certificate scoped to a cluster
func CreateClusterCertificate(c *gin.Context) {
	clusterID := c.Param("clusterID")

	var req models.CreateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cert, err := services.CreateClusterCertificate(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.CertificateResponse{
		ID:          cert.ID,
		Name:        cert.Name,
		Description: cert.Description,
		Scope:       cert.Scope,
		ClusterID:   cert.ClusterID,
		EnvID:       derefCertString(cert.EnvID),
		CreatedAt:   cert.CreatedAt,
	})
}

// CreateEnvCertificate creates a certificate scoped to an environment
func CreateEnvCertificate(c *gin.Context) {
	envID := c.Param("envID")

	var req models.CreateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cert, err := services.CreateEnvCertificate(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.CertificateResponse{
		ID:          cert.ID,
		Name:        cert.Name,
		Description: cert.Description,
		Scope:       cert.Scope,
		ClusterID:   cert.ClusterID,
		EnvID:       derefCertString(cert.EnvID),
		CreatedAt:   cert.CreatedAt,
	})
}

// UpdateCertificate updates an existing certificate
func UpdateCertificate(c *gin.Context) {
	certID := c.Param("certID")

	var req models.UpdateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cert, err := services.UpdateCertificate(certID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.CertificateResponse{
		ID:          cert.ID,
		Name:        cert.Name,
		Description: cert.Description,
		Scope:       cert.Scope,
		ClusterID:   cert.ClusterID,
		EnvID:       derefCertString(cert.EnvID),
		CreatedAt:   cert.CreatedAt,
	})
}

// DeleteCertificate deletes a certificate by ID
func DeleteCertificate(c *gin.Context) {
	certID := c.Param("certID")

	if err := services.DeleteCertificate(certID); err != nil {
		if errors.Is(err, services.ErrCertificateInUse) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func derefCertString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
