import client from './client'
import { type PaginationParams, type PaginationResponse, type SimpleResponse } from './pagination'

export interface Cluster {
  id: string
  slug: string
  name: string
  description?: string
  enabled?: boolean
  kube_config?: string
  gateway_ip?: string
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
  gateway_ip?: string
}

export interface PingClusterRequest {
  kube_config: string
}

export interface PingClusterResponse {
  message: string
}

export interface UpdateClusterCredentialsRequest {
  kube_config: string
  gateway_ip?: string
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

export const clustersApi = {
  list: async (params?: PaginationParams) => {
    return client.get('/v1/clusters', { params }) as Promise<{ items: Cluster[], pagination: PaginationResponse }>
  },

  listSimple: async () => {
    return client.get('/v1/clusters/simple') as Promise<SimpleResponse[]>
  },

  listPublic: async () => {
    return client.get('/v1/clusters/public') as Promise<Cluster[]>
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

  getPublic: async (id: string) => {
    return client.get(`/v1/clusters/${id}/public`) as Promise<Cluster>
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

  listIntegrations: async (clusterId: string) => {
    return client.get(`/v1/clusters/${clusterId}/integrations`) as Promise<ClusterIntegration[]>
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

  prometheusQuery: async (clusterId: string, query: string, time?: string) => {
    const params = new URLSearchParams({ query })
    if (time) params.set('time', time)
    return client.get(`/v1/clusters/${clusterId}/prometheus/query?${params}`)
  },

  prometheusQueryRange: async (clusterId: string, query: string, start: string, end: string, step: string) => {
    const params = new URLSearchParams({ query, start, end, step })
    return client.get(`/v1/clusters/${clusterId}/prometheus/query_range?${params}`)
  },

  listNamespaces: async (id: string) => {
    return client.get(`/v1/clusters/${id}/namespaces`) as Promise<string[]>
  },

  listServices: async (id: string, namespace: string) => {
    return client.get(`/v1/clusters/${id}/services?namespace=${namespace}`) as Promise<string[]>
  },

  listStorageClasses: async (clusterId: string) => {
    return client.get(`/v1/clusters/${clusterId}/storage-classes`) as Promise<Array<{
      name: string
      provisioner: string
      isDefault: boolean
    }>>
  },

  // Extension catalog (admin-managed, global)
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

  uninstallExtension: async (clusterId: string, clusterExtensionId: string) =>
    client.delete(`/v1/clusters/${clusterId}/extensions/${clusterExtensionId}`) as Promise<ClusterExtension>,

  // Gateway API status
  getGatewayAPIStatus: async (clusterId: string) => {
    return client.get(`/v1/clusters/${clusterId}/gateway-api-status`) as Promise<GatewayAPIStatus>
  },
}

// Extension types

// Extension catalog entry (global, admin-managed)
export interface Extension {
  id: string
  name: string
  display_name?: string
  description?: string
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

// Request to create a new extension in catalog
export interface CreateExtensionRequest {
  name: string
  display_name?: string
  description?: string
  oci_url: string
  icon_url?: string
}

// Request to update an extension in catalog
export interface UpdateExtensionRequest {
  display_name?: string
  description?: string
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
  release_name: string
  namespace: string
  version: string
  status: string
}
