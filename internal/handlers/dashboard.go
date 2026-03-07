package handlers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/services"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetDashboardStats(c *gin.Context) {
	userRole := api.GetUserRole(c)
	projectID := c.Query("project_id")

	if userRole == app.UserRoleAdmin {
		stats, err := services.GetAdminDashboardStats()
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

	stats, err := services.GetUserDashboardStats(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, stats)
}

func GetDashboardEnvironments(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		api.Error(c, http.StatusBadRequest, app.ErrBadRequest)
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
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
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

	integration, err := services.GetClusterIntegrationByType(clusterID, entities.IntegrationTypePrometheus)
	if err != nil {
		api.Error(c, http.StatusNotFound, fmt.Errorf("prometheus integration not configured for this cluster"))
		return
	}

	query := c.Query("query")
	queryTime := c.Query("time")

	if query == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("query parameter is required"))
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

	integration, err := services.GetClusterIntegrationByType(clusterID, entities.IntegrationTypePrometheus)
	if err != nil {
		api.Error(c, http.StatusNotFound, fmt.Errorf("prometheus integration not configured for this cluster"))
		return
	}

	query := c.Query("query")
	start := c.Query("start")
	end := c.Query("end")
	step := c.Query("step")

	if query == "" || start == "" || end == "" || step == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("query, start, end, and step parameters are required"))
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
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
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

	log.Printf("Prometheus request: %s", urlStr)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return nil, err
	}

	if integration.ServiceName == "" {
		if integration.Token != "" {
			req.Header.Set("Authorization", "Bearer "+integration.Token)
		} else if integration.Username != "" && integration.Password != "" {
			req.SetBasicAuth(integration.Username, integration.Password)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Prometheus request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Prometheus returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to parse response JSON: %v, body: %s", err, string(body))
		return nil, err
	}

	return result, nil
}
