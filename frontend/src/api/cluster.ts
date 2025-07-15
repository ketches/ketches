import api from '@/api/axios';
import type { clusterExtensionModel, clusterModel, clusterNodeModel, clusterNodeRefModel, clusterNodeTaintsModel, clusterRefModel, createClusterModel, installClusterExtensionModel, updateClusterModel } from '@/types/cluster';
import type { QueryAndPagedRequest } from '@/types/common.ts';

export async function listClusters(filter: QueryAndPagedRequest): Promise<{ total: number, records: clusterModel[] }> {
    const response = await api.get('/clusters', {
        params: filter
    });
    return response.data as { total: number, records: clusterModel[] };
}

export async function fetchClusterRefs(): Promise<clusterRefModel[]> {
    const response = await api.get(`/clusters/refs`)
    return response.data as clusterRefModel[];
}

export async function getCluster(clusterID: string): Promise<clusterModel> {
    const response = await api.get(`/clusters/${clusterID}`);
    return response.data as clusterModel;
}

export async function getClusterRef(clusterID: string): Promise<clusterRefModel> {
    const response = await api.get(`/clusters/${clusterID}/ref`);
    return response.data as clusterRefModel;
}

export async function createCluster(model: createClusterModel): Promise<clusterModel> {
    const response = await api.post('/clusters', model);
    return response.data as clusterModel;
}

export async function updateCluster(clusterID: string, model: updateClusterModel): Promise<clusterModel> {
    const response = await api.put(`/clusters/${clusterID}`, model);
    return response.data as clusterModel;
}

export async function deleteCluster(clusterID: string): Promise<boolean> {
    await api.delete(`/clusters/${clusterID}`);
    return true;
}

export async function enableCluster(clusterID: string): Promise<clusterModel> {
    const response = await api.post(`/clusters/${clusterID}/enable`);
    return response.data as clusterModel;
}

export async function disableCluster(clusterID: string): Promise<clusterModel> {
    const response = await api.post(`/clusters/${clusterID}/disable`);
    return response.data as clusterModel;
}

export async function pingClusterKubeConfig(kubeConfig: string): Promise<boolean> {
    const response = await api.post('/clusters/ping', { kubeConfig });
    return response.data as boolean;
}

export async function listClusterNodes(clusterID: string): Promise<clusterNodeModel[]> {
    const response = await api.get(`/clusters/${clusterID}/nodes`);
    return response.data as clusterNodeModel[];
}

export async function listClusterNodeRefs(clusterID: string): Promise<clusterNodeRefModel[]> {
    const response = await api.get(`/clusters/${clusterID}/nodes/refs`);
    return response.data as clusterNodeRefModel[];
}

export async function getClusterNode(clusterID: string, nodeName: string): Promise<clusterNodeModel> {
    const response = await api.get(`/clusters/${clusterID}/nodes/${nodeName}`);
    return response.data as clusterNodeModel;
}

export async function listClusterNodeLabels(clusterID: string): Promise<string[]> {
    const response = await api.get(`/clusters/${clusterID}/nodes/labels`);
    return response.data as string[];
}

export async function listClusterNodeTaints(clusterID: string): Promise<clusterNodeTaintsModel[]> {
    const response = await api.get(`/clusters/${clusterID}/nodes/taints`);
    return response.data as clusterNodeTaintsModel[];
}

export async function checkClusterExtensionFeatureEnabled(clusterID: string): Promise<boolean> {
    const response = await api.get(`/clusters/${clusterID}/extensions/feature-enabled`);
    return response.data as boolean;
}

export async function enableClusterExtension(clusterID: string): Promise<void> {
    const response = await api.post(`/clusters/${clusterID}/extensions/enable`);
}

export async function listClusterExtensions(clusterID: string): Promise<Record<string, clusterExtensionModel>> {
    const response = await api.get(`/clusters/${clusterID}/extensions`);
    return response.data as Record<string, clusterExtensionModel>;
}

export async function installClusterExtension(clusterID: string, model: installClusterExtensionModel): Promise<boolean> {
    await api.post(`/clusters/${clusterID}/extensions/install`, model);
    return true;
}

export async function uninstallClusterExtension(clusterID: string, extensionName: string): Promise<boolean> {
    await api.delete(`/clusters/${clusterID}/extensions/${extensionName}`);
    return true;
}

export async function getClusterExtensionValues(clusterID: string, extensionName: string, version: string): Promise<string> {
    const response = await api.get(`/clusters/${clusterID}/extensions/${extensionName}/values/${version}`);
    return response.data as string;
}
