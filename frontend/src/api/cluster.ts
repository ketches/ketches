import api from '@/api/axios';
import type { clusterModel, clusterNodeModel, clusterNodeRefModel, clusterRefModel, createClusterModel, updateClusterModel } from '@/types/cluster';
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
