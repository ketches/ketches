import api from '@/api/axios';
import type { QueryAndPagedRequest } from '@/types/common';
import type { envCreateModel, envModel, envRefModel, updateEnvModel } from '@/types/env';

export async function listEnvs(projectID: string, filter: QueryAndPagedRequest): Promise<{ total: number, records: envModel[] }> {
    const response = await api.get(`/envs`, {
        params: {
            ...filter,
            projectID
        }
    })
    return response.data as { total: number, records: envModel[] }
}

export async function fetchEnvRefs(projectID: string): Promise<envRefModel[]> {
    const response = await api.get(`/envs/refs`, {
        params: {
            projectID
        }
    })
    return response.data as envRefModel[]
}

export async function getEnv(envID: string): Promise<envModel> {
    const response = await api.get(`/envs/${envID}`)
    return response.data as envModel
}

export async function getEnvRef(envID: string): Promise<envRefModel> {
    const response = await api.get(`/envs/${envID}/ref`)
    return response.data as envRefModel
}

export async function createEnv(model: envCreateModel): Promise<envModel> {
    const response = await api.post(`/envs`, model)
    return response.data as envModel
}

export async function updateEnv(envID: string, model: updateEnvModel): Promise<envModel> {
    const response = await api.put(`/envs/${envID}`, model)
    return response.data as envModel
}

export async function deleteEnv(envID: string): Promise<boolean> {
    await api.delete(`/envs/${envID}`)
    return true
}
