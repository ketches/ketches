package models

// ImageMetadataResponse is the response body for the GET image-metadata endpoint.
type ImageMetadataResponse struct {
	Env          []ImageEnvVar     `json:"env"`
	Volumes      []string          `json:"volumes"`
	ExposedPorts []ImagePortInfo   `json:"exposed_ports"`
	HealthCheck  *ImageHealthCheck `json:"health_check,omitempty"`
}

// ImageEnvVar is an environment variable key-value pair parsed from an image.
type ImageEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ImagePortInfo represents a port exposed by the image.
type ImagePortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "TCP" or "UDP"
}

// ImageHealthCheck holds the health check configuration embedded in the image.
type ImageHealthCheck struct {
	Test     []string `json:"test"`
	Interval int      `json:"interval"` // seconds
	Timeout  int      `json:"timeout"`  // seconds
	Retries  int      `json:"retries"`
}
