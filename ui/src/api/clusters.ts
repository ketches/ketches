import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface SimpleCluster {
  id: string
  name: string
  slug: string
  description: string
  connection_status?: string
  enabled: boolean
}

export interface Cluster {
	id: string
	slug: string
	name: string
	description?: string
	enabled?: boolean
	api_server?: string
	has_kube_config?: boolean
	gateway_host?: string
	has_prometheus_integration?: boolean
	connection_status?: string
	connection_status_reason?: string
	last_checked_at?: string
	created_at?: string
}

export interface CreateClusterRequest {
	slug: string
	name: string
	description?: string
	kube_config: string
	gateway_host?: string
}

export interface PingClusterRequest {
  kube_config: string
}

export interface PingClusterResponse {
  message: string
}

export interface UpdateClusterCredentialsRequest {
	kube_config?: string
	gateway_host?: string
}

export interface K8sNode {
  metadata: {
    name: string
    creationTimestamp: string
    labels: Record<string, string>
    annotations: Record<string, string>
  }
  spec: {
    unschedulable?: boolean
    taints?: Array<{
      key: string
      value?: string
      effect: string
    }>
  }
  status: {
    nodeInfo: {
      machineID: string
      systemUUID: string
      bootID: string
      kernelVersion: string
      osImage: string
      containerRuntimeVersion: string
      kubeletVersion: string
      kubeProxyVersion: string
      operatingSystem: string
      architecture: string
    }
    capacity: {
      cpu: string
      memory: string
      pods: string
      "ephemeral-storage": string
    }
    allocatable: {
      cpu: string
      memory: string
      pods: string
      "ephemeral-storage": string
    }
    addresses: Array<{
      type: string
      address: string
    }>
    conditions: Array<{
      type: string
      status: string
      reason?: string
      message?: string
    }>
  }
}

export interface CreateClusterGatewayProviderRequest {
  display_name?: string
  gateway_class_name: string
  controller_name: string
  make_default?: boolean
}

export interface ClusterGatewayProvider {
  id: string
  cluster_id: string
  source_type: string
  display_name: string
  gateway_class_name: string
  controller_name: string
  extension_id?: string
  cluster_extension_id?: string
  is_default: boolean
}

export interface GatewayClassSummary {
  name: string
  controller_name: string
  accepted: boolean
  is_default: boolean
}

export interface UpdateClusterGatewayClassRequest {
  gateway_class_name: string
  gateway_controller_name?: string
  management_mode?: string
}

export interface ClusterServicePort {
  name?: string
  protocol: string
  port: number
  target_port: string
  node_port?: number
}

export interface ClusterService {
  name: string
  ports: ClusterServicePort[]
}

export interface PrometheusMetricResult {
  metric: Record<string, string>
  value?: [number, string]
  values?: [number, string][]
}

export interface PrometheusResponse {
  resultType: string
  result: PrometheusMetricResult[]
}

function buildProjectScopedParams(projectId?: string) {
  if (!projectId) {
    return undefined
  }

  return { project_id: projectId }
}

export const clustersApi = {
  list: async (params?: PaginationParams) => {
    return client.get('/v1/clusters', { params }) as Promise<{ items: Cluster[], pagination: PaginationResponse }>
  },

  listSimple: async () => {
    return client.get('/v1/clusters/simple') as Promise<SimpleCluster[]>
  },

  listPublic: async (projectId?: string) => {
    return client.get('/v1/clusters/public', {
      params: buildProjectScopedParams(projectId),
    }) as Promise<Cluster[]>
  },

  create: async (data: CreateClusterRequest) => {
    return client.post('/v1/clusters', data) as Promise<Cluster>
  },

  ping: async (data: PingClusterRequest) => {
    return client.post('/v1/clusters/ping', data) as Promise<PingClusterResponse>
  },

  get: async (id: string) => {
    return client.get(`/v1/clusters/${id}`) as Promise<Cluster>
  },

  getPublic: async (id: string, projectId?: string) => {
    return client.get(`/v1/clusters/${id}/public`, {
      params: buildProjectScopedParams(projectId),
    }) as Promise<Cluster>
  },

  delete: async (id: string) => {
    return client.delete(`/v1/clusters/${id}`)
  },

  listNodes: async (id: string) => {
    return client.get(`/v1/clusters/${id}/nodes`) as Promise<K8sNode[]>
  },

  getNode: async (clusterId: string, nodeName: string) => {
    return client.get(`/v1/clusters/${clusterId}/nodes/${nodeName}`) as Promise<K8sNode>
  },

  updateNodeLabels: async (clusterId: string, nodeName: string, labels: Record<string, string>) => {
    return client.patch(`/v1/clusters/${clusterId}/nodes/${nodeName}/labels`, { labels })
  },

  updateNodeAnnotations: async (clusterId: string, nodeName: string, annotations: Record<string, string>) => {
    return client.patch(`/v1/clusters/${clusterId}/nodes/${nodeName}/annotations`, { annotations })
  },

  updateNodeTaints: async (clusterId: string, nodeName: string, taints: Array<{ taint_key: string, taint_value?: string, effect: string }>) => {
    return client.patch(`/v1/clusters/${clusterId}/nodes/${nodeName}/taints`, { taints })
  },

  cordonNode: async (clusterId: string, nodeName: string, cordon: boolean, evict_pods: boolean = false) => {
    return client.patch(`/v1/clusters/${clusterId}/nodes/${nodeName}/cordon`, { cordon, evict_pods })
  },

  update: async (id: string, data: Partial<Cluster>) => {
    return client.patch(`/v1/clusters/${id}/basic`, data) as Promise<Cluster>
  },

  checkConnectivity: async (id: string) => {
    return client.post(`/v1/clusters/${id}/check-connectivity`)
  },

  updateCredentials: async (id: string, data: UpdateClusterCredentialsRequest) => {
    return client.patch(`/v1/clusters/${id}/credentials`, data) as Promise<Cluster>
  },

  listIntegrations: async (clusterId: string, projectId?: string) => {
    return client.get(`/v1/clusters/${clusterId}/integrations`, {
      params: buildProjectScopedParams(projectId),
    }) as Promise<ClusterIntegration[]>
  },

  createIntegration: async (clusterId: string, data: CreateClusterIntegrationRequest) => {
    return client.post(`/v1/clusters/${clusterId}/integrations`, data) as Promise<ClusterIntegration>
  },

  updateIntegration: async (clusterId: string, integrationId: string, data: UpdateClusterIntegrationRequest) => {
    return client.put(`/v1/clusters/${clusterId}/integrations/${integrationId}`, data) as Promise<ClusterIntegration>
  },

  deleteIntegration: async (clusterId: string, integrationId: string) => {
    return client.delete(`/v1/clusters/${clusterId}/integrations/${integrationId}`)
  },

  prometheusQuery: async (clusterId: string, query: string, time?: string, projectId?: string) => {
    const params = new URLSearchParams({ query })
    if (time) params.set('time', time)
    if (projectId) params.set('project_id', projectId)
    return client.get(`/v1/clusters/${clusterId}/prometheus/query?${params}`)
  },

  prometheusQueryRange: async (clusterId: string, query: string, start: string, end: string, step: string, projectId?: string) => {
    const params = new URLSearchParams({ query, start, end, step })
    if (projectId) params.set('project_id', projectId)
    return client.get(`/v1/clusters/${clusterId}/prometheus/query_range?${params}`) as Promise<PrometheusResponse>
  },

  listNamespaces: async (id: string) => {
    return client.get(`/v1/clusters/${id}/namespaces`) as Promise<string[]>
  },

  listGatewayClasses: async (id: string) => {
    return client.get(`/v1/clusters/${id}/gateway-classes`) as Promise<GatewayClassSummary[]>
  },

  listGatewayProviders: async (id: string) => {
    return client.get(`/v1/clusters/${id}/gateway-providers`) as Promise<ClusterGatewayProvider[]>
  },

  createGatewayProvider: async (id: string, data: CreateClusterGatewayProviderRequest) => {
    return client.post(`/v1/clusters/${id}/gateway-providers`, data) as Promise<ClusterGatewayProvider>
  },

  deleteGatewayProvider: async (clusterId: string, providerId: string) => {
    return client.delete(`/v1/clusters/${clusterId}/gateway-providers/${providerId}`)
  },

  updateDefaultGatewayClass: async (id: string, data: UpdateClusterGatewayClassRequest) => {
    return client.put(`/v1/clusters/${id}/default-gateway-class`, data) as Promise<Cluster>
  },

  listServices: async (id: string, namespace: string) => {
    return client.get(`/v1/clusters/${id}/services?namespace=${namespace}`) as Promise<string[]>
  },

  listServicesWithPorts: async (id: string, namespace: string) => {
    return client.get(`/v1/clusters/${id}/services?namespace=${namespace}&with_ports=true`) as Promise<ClusterService[]>
  },

  listStorageClasses: async (clusterId: string, projectId?: string) => {
    return client.get(`/v1/clusters/${clusterId}/storage-classes`, {
      params: buildProjectScopedParams(projectId),
    }) as Promise<Array<{
      name: string
      provisioner: string
      isDefault: boolean
    }>>
  },

  // Extension (admin-managed, global)
  listExtensions: async () =>
    client.get('/v1/extensions') as Promise<Extension[]>,

  createExtension: async (data: CreateExtensionRequest) =>
    client.post('/v1/extensions', data) as Promise<Extension>,

  deleteExtension: async (extensionId: string) =>
    client.delete(`/v1/extensions/${extensionId}`),

  updateExtension: async (extensionId: string, data: UpdateExtensionRequest) =>
    client.put(`/v1/extensions/${extensionId}`, data) as Promise<Extension>,

  getExtensionVersions: async (extensionId: string) =>
    client.get(`/v1/extensions/${extensionId}/versions`) as Promise<ExtensionVersionInfo[]>,

  getExtensionValues: async (extensionId: string, version: string) =>
    client.get(`/v1/extensions/${extensionId}/versions/${version}/values`) as Promise<{ values: string }>,

  getExtensionInstalledClusters: async (extensionId: string) =>
    client.get(`/v1/extensions/${extensionId}/installed-clusters`) as Promise<InstalledCluster[]>,

  // Cluster extensions (per-cluster installed)
  listClusterExtensions: async (clusterId: string) =>
    client.get(`/v1/clusters/${clusterId}/extensions`) as Promise<ClusterExtension[]>,

  getClusterExtension: async (clusterId: string, clusterExtensionId: string) =>
    client.get(`/v1/clusters/${clusterId}/extensions/${clusterExtensionId}`) as Promise<ClusterExtension>,

  installExtension: async (clusterId: string, data: InstallExtensionRequest) =>
    client.post(`/v1/clusters/${clusterId}/extensions`, data) as Promise<ClusterExtension>,

  upgradeExtension: async (clusterId: string, clusterExtensionId: string, data: UpgradeExtensionRequest) =>
    client.put(`/v1/clusters/${clusterId}/extensions/${clusterExtensionId}`, data) as Promise<ClusterExtension>,

  retryExtension: async (clusterId: string, clusterExtensionId: string, data?: RetryClusterExtensionRequest) =>
    client.post(`/v1/clusters/${clusterId}/extensions/${clusterExtensionId}/retry`, data ?? {}) as Promise<ClusterExtension>,

  uninstallExtension: async (clusterId: string, clusterExtensionId: string) =>
    client.delete(`/v1/clusters/${clusterId}/extensions/${clusterExtensionId}`) as Promise<ClusterExtension>,

  // Gateway API status
  getGatewayAPIStatus: async (clusterId: string, projectId?: string) => {
    return client.get(`/v1/clusters/${clusterId}/gateway-api-status`, {
      params: buildProjectScopedParams(projectId),
    }) as Promise<GatewayAPIStatus>
  },
}

// Extension types

// Extension entry (global, admin-managed)
export interface Extension {
  id: string
  name: string
  display_name?: string
  description?: string
  capabilities?: string[]
  metadata?: Record<string, unknown>
  oci_url: string
  icon_url?: string
  install_count: number
  builtin: boolean
  created_at: string
}

export interface ExtensionVersionInfo {
  version: string
}

// Installed extension on a specific cluster
export interface ClusterExtension {
  id: string
  cluster_id: string
  extension_id: string
  name: string
  namespace: string
  release_name: string
  version: string
  values?: string
  status: string
  phase?: string
  error_message?: string
  created_at: string
}

// Request to install an extension on a cluster
export interface InstallExtensionRequest {
  extension_id: string
  release_name: string
  namespace?: string
  version?: string
  create_namespace?: boolean
  values?: string
}

// Request to upgrade a cluster extension
export interface UpgradeExtensionRequest {
  version?: string
  values?: string
}

export interface RetryClusterExtensionRequest {
  name?: string
  values?: string
}

// Request to create a new extension
export interface CreateExtensionRequest {
  name: string
  display_name?: string
  description?: string
  capabilities?: string[]
  metadata?: Record<string, unknown>
  oci_url: string
  icon_url?: string
}

// Request to update an extension
export interface UpdateExtensionRequest {
  display_name?: string
  description?: string
  capabilities?: string[]
  metadata?: Record<string, unknown>
  oci_url?: string
  icon_url?: string
}

export type IntegrationType = 'prometheus' | 'grafana' | 'loki' | 'alertmanager'

export interface ClusterIntegration {
	id: string
	cluster_id: string
  integration_type: IntegrationType
  name: string
  endpoint: string
  namespace?: string
  service_name?: string
	service_port?: number
	username?: string
	has_password?: boolean
	has_token?: boolean
	has_ca_cert?: boolean
	skip_tls_verify: boolean
	enabled: boolean
	created_at: string
}

export interface CreateClusterIntegrationRequest {
  integration_type: IntegrationType
  name: string
  endpoint?: string
  namespace?: string
  service_name?: string
  service_port?: number
  username?: string
  password?: string
  token?: string
  ca_cert?: string
  skip_tls_verify?: boolean
  enabled?: boolean
}

export interface UpdateClusterIntegrationRequest {
	name?: string
	endpoint?: string
  namespace?: string
  service_name?: string
  service_port?: number
	username?: string
	password?: string
	token?: string
	ca_cert?: string
	clear_password?: boolean
	clear_token?: boolean
	clear_ca_cert?: boolean
	skip_tls_verify?: boolean
	enabled?: boolean
}

// GatewayAPIStatus indicates whether the Gateway API is installed on a cluster.
export interface GatewayAPIStatus {
  installed: boolean
}

export interface InstalledCluster {
  cluster_id: string
  cluster_name: string
  name: string
  release_name: string
  namespace: string
  version: string
  status: string
}
