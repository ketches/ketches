import api from '@/api/axios';
import type { appModel, appRefModel, createAppModel } from '@/types/app';
import type { QueryAndPagedRequest } from '@/types/common';
import type { envModel, envRefModel, updateEnvModel } from '@/types/env';

export async function getEnv(envID: string): Promise<envModel> {
    const response = await api.get(`/envs/${envID}`)
    return response.data as envModel
}

export async function getEnvRef(envID: string): Promise<envRefModel> {
    const response = await api.get(`/envs/${envID}/ref`)
    return response.data as envRefModel
}

export async function updateEnv(envID: string, model: updateEnvModel): Promise<envModel> {
    const response = await api.put(`/envs/${envID}`, model)
    return response.data as envModel
}

export async function deleteEnv(envID: string): Promise<boolean> {
    await api.delete(`/envs/${envID}`)
    return true
}

export async function listApps(envID: string, filter: QueryAndPagedRequest): Promise<{ total: number, records: appModel[] }> {
    const response = await api.get(`/envs/${envID}/apps`, {
        params: filter,
    })
    return response.data as { total: number, records: appModel[] }
}

export async function fetchAppRefs(envID: string): Promise<appRefModel[]> {
    const response = await api.get(`/envs/${envID}/apps/refs`)
    return response.data as appRefModel[]
}

export async function createApp(envID: string, model: createAppModel): Promise<appModel> {
    const response = await api.post(`/envs/${envID}/apps`, model)
    return response.data as appModel
}