package handlers

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/internal/services"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetDashboardStats(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	projectID := queryProjectID(c)

	if claims.Role == app.UserRoleAdmin {
		stats, err := services.GetAdminDashboardStats(projectID)
		if err != nil {
			api.Error(c, http.StatusInternalServerError, err)
			return
		}
		api.Success(c, stats)
		return
	}

	if projectID == "" {
		api.Error(c, http.StatusBadRequest, app.ErrBadRequest)
		return
	}

	if requireProjectAccess(c, projectID) == nil {
		return
	}

	stats, err := services.GetUserDashboardStats(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, stats)
}

func GetDashboardEnvironments(c *gin.Context) {
	projectID := queryProjectID(c)
	if projectID == "" {
		api.Error(c, http.StatusBadRequest, app.ErrBadRequest)
		return
	}

	if requireProjectAccess(c, projectID) == nil {
		return
	}

	envs, err := services.GetProjectEnvironmentsWithNamespaces(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, envs)
}

func getIntegrationEndpoint(integration *entities.ClusterIntegration) (string, error) {
	if integration.ServiceName != "" {
		cluster, err := services.GetCluster(integration.ClusterID)
		if err != nil {
			return "", err
		}
		plaintextKubeConfig, err := secrets.DecryptString(cluster.KubeConfig)
		if err != nil {
			return "", err
		}
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
		if err != nil {
			return "", err
		}
		host := strings.TrimSuffix(config.Host, "/")
		port := integration.ServicePort
		if port == 0 {
			port = 80
		}
		return fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s:%d/proxy", host, integration.Namespace, integration.ServiceName, port), nil
	}
	return strings.TrimSuffix(integration.Endpoint, "/"), nil
}

func ProxyPrometheusQuery(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	integration, err := services.GetClusterIntegrationByType(clusterID, entities.IntegrationTypePrometheus)
	if err != nil {
		api.Error(c, http.StatusNotFound, app.NewErrorf("prometheus integration not configured for this cluster"))
		return
	}

	query := c.Query("query")
	queryTime := c.Query("time")

	if query == "" {
		api.Error(c, http.StatusBadRequest, app.NewErrorf("query parameter is required"))
		return
	}

	endpoint, err := getIntegrationEndpoint(integration)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	fullURL := endpoint + "/api/v1/query"

	promURL, err := url.Parse(fullURL)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	params := url.Values{}
	params.Set("query", query)
	if queryTime != "" {
		params.Set("time", queryTime)
	}
	promURL.RawQuery = params.Encode()

	result, err := executePrometheusRequest(promURL.String(), integration)
	if err != nil {
		api.Error(c, http.StatusBadGateway, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func ProxyPrometheusQueryRange(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	integration, err := services.GetClusterIntegrationByType(clusterID, entities.IntegrationTypePrometheus)
	if err != nil {
		api.Error(c, http.StatusNotFound, app.NewErrorf("prometheus integration not configured for this cluster"))
		return
	}

	query := c.Query("query")
	start := c.Query("start")
	end := c.Query("end")
	step := c.Query("step")

	if query == "" || start == "" || end == "" || step == "" {
		api.Error(c, http.StatusBadRequest, app.NewErrorf("query, start, end, and step parameters are required"))
		return
	}

	endpoint, err := getIntegrationEndpoint(integration)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	fullURL := endpoint + "/api/v1/query_range"

	promURL, err := url.Parse(fullURL)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)
	promURL.RawQuery = params.Encode()

	result, err := executePrometheusRequest(promURL.String(), integration)
	if err != nil {
		api.Error(c, http.StatusBadGateway, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func executePrometheusRequest(urlStr string, integration *entities.ClusterIntegration) (any, error) {
	var transport http.RoundTripper
	var err error

	if integration.ServiceName != "" {
		cluster, err := services.GetCluster(integration.ClusterID)
		if err != nil {
			return nil, err
		}
		plaintextKubeConfig, err := secrets.DecryptString(cluster.KubeConfig)
		if err != nil {
			return nil, err
		}
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
		if err != nil {
			return nil, err
		}
		transport, err = rest.TransportFor(config)
		if err != nil {
			return nil, err
		}
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: integration.SkipTLSVerify,
			},
		}
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	slog.Debug(fmt.Sprintf("Prometheus request: %s", urlStr))

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create request: %v", err))
		return nil, err
	}

	if integration.ServiceName == "" {
		if integration.Token != "" {
			plaintextToken, err := secrets.DecryptString(integration.Token)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+plaintextToken)
		} else if integration.Username != "" && integration.Password != "" {
			plaintextPassword, err := secrets.DecryptString(integration.Password)
			if err != nil {
				return nil, err
			}
			req.SetBasicAuth(integration.Username, plaintextPassword)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error(fmt.Sprintf("Prometheus request failed: %v", err))
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to read response body: %v", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Info(fmt.Sprintf("Prometheus returned status %d: %s", resp.StatusCode, string(body)))
		return nil, app.NewErrorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error(fmt.Sprintf("Failed to parse response JSON: %v, body: %s", err, string(body)))
		return nil, err
	}

	return result, nil
}
