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
