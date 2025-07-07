export interface clusterModel {
    clusterID: string;
    slug: string;
    displayName: string;
    description?: string;
    kubeConfig?: string;
    readyNodeCount?: number;
    nodeCount?: number;
    serverVersion?: string;
    connectable?: boolean;
    enabled: boolean;
}

export interface clusterRefModel {
    clusterID: string;
    slug: string;
    displayName: string;
}

export interface createClusterModel {
    slug: string
    displayName: string
    kubeConfig: string
    gatewayIP?: string
    description?: string
}

export interface updateClusterModel {
    displayName: string,
    kubeConfig: string
    description?: string,
}

export interface clusterNodeModel {
    nodeName: string;
    roles: string[];
    createdAt: string;
    version: string;
    internalIP: string;
    externalIP: string;
    osImage: string;
    kernelVersion: string;
    operatingSystem: string;
    architecture: string;
    containerRuntimeVersion: string;
    kubeletVersion: string;
    podCIDR: string;
    ready: boolean;
    clusterID: string;
}

export interface clusterNodeRefModel {
    nodeName: string
    nodeIP: string
    clusterID: string
    clusterSlug: string
    clusterDisplayName: string
}

export interface clusterNodeTaintsModel {
    key: string;
    values: string[];
}
