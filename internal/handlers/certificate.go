package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
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
		res = append(res, toCertificateResponse(&cert))
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
		res = append(res, toCertificateResponse(&cert))
	}

	api.Success(c, models.ListCertificatesResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

// GetCertificate returns a certificate scoped to the environment in the URL.
func GetCertificate(c *gin.Context) {
	envID := c.Param("envID")
	certID := c.Param("certID")

	cert, err := services.GetCertificate(envID, certID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, toCertificateResponse(cert))
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

	api.Created(c, toCertificateResponse(cert))
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

	api.Created(c, toCertificateResponse(cert))
}

// UpdateCertificate updates an environment certificate.
func UpdateCertificate(c *gin.Context) {
	envID := c.Param("envID")
	certID := c.Param("certID")

	var req models.UpdateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cert, err := services.UpdateCertificate(envID, certID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toCertificateResponse(cert))
}

// DeleteCertificate deletes an environment certificate.
func DeleteCertificate(c *gin.Context) {
	envID := c.Param("envID")
	certID := c.Param("certID")

	if err := services.DeleteCertificate(envID, certID); err != nil {
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

func toCertificateResponse(cert *entities.Certificate) models.CertificateResponse {
	return models.CertificateResponse{
		ID:            cert.ID,
		Name:          cert.Name,
		Description:   cert.Description,
		Scope:         cert.Scope,
		ClusterID:     cert.ClusterID,
		EnvID:         derefCertString(cert.EnvID),
		HasPrivateKey: cert.Key != "",
		CreatedAt:     cert.CreatedAt,
	}
}
