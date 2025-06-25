import api from '@/api/axios';
import type { QueryAndPagedRequest } from '@/types/common';
import type { envCreateModel, envModel, envRefModel } from '@/types/env';
import type { createProjectModel, projectMemberModel, projectModel, projectRefModel } from '@/types/project';
import type { userRefModel } from '@/types/user';

export async function listProjects(filter: QueryAndPagedRequest): Promise<projectModel[]> {
    const response = await api.get('/projects', {
        params: filter
    })
    return response.data as projectModel[]
}

export async function fetchProjectRefs(): Promise<projectRefModel[]> {
    const response = await api.get('/projects/refs')
    return response.data as projectRefModel[]
}

export async function getProject(projectID: string): Promise<projectModel> {
    const response = await api.get(`/projects/${projectID}`)
    return response.data as projectModel
}

export async function getProjectRef(projectID: string): Promise<projectRefModel> {
    const response = await api.get(`/projects/${projectID}/ref`)
    return response.data as projectRefModel
}

export async function createProject(model: createProjectModel): Promise<projectModel> {
    const response = await api.post('/projects', model)
    return response.data as projectModel
}

export async function updateProject(projectID: string, displayName: string, description: string): Promise<projectModel> {
    const response = await api.put(`/projects/${projectID}`, {
        displayName,
        description
    })
    return response.data as projectModel
}

export async function deleteProject(projectID: string): Promise<boolean> {
    await api.delete(`/projects/${projectID}`)
    return true
}

export async function listProjectMembers(projectID: string, filter: QueryAndPagedRequest): Promise<{ total: number, records: projectMemberModel[] }> {
    const response = await api.get(`/projects/${projectID}/members`, {
        params: filter
    })
    return response.data as { total: number, records: projectMemberModel[] }
}

export async function listAddableProjectMembers(projectID: string): Promise<userRefModel[]> {
    const response = await api.get(`/projects/${projectID}/members/addable`)
    return response.data as userRefModel[]
}

export async function addProjectMember(projectID: string, userIDs: string[], projectRole: string): Promise<boolean> {
    await api.post(`/projects/${projectID}/members`, {
        projectMembers: userIDs.map(userID => ({ userID, projectRole }))
    })
    return true
}

export async function removeProjectMembers(projectID: string, userIDs: string[]): Promise<boolean> {
    await api.delete(`/projects/${projectID}/members`, {
        data: {
            userIDs: userIDs,
        },
    })
    return true
}

export async function updateProjectMemberRole(projectID: string, userID: string, projectRole: string): Promise<projectMemberModel> {
    const response = await api.put(`/projects/${projectID}/members/${userID}`, {
        projectRole
    })
    return response.data as projectMemberModel
}

export async function listEnvs(projectID: string, filter: QueryAndPagedRequest): Promise<{ total: number, records: envModel[] }> {
    const response = await api.get(`/projects/${projectID}/envs`, {
        params: filter,
    })
    return response.data as { total: number, records: envModel[] }
}

export async function fetchEnvRefs(projectID: string): Promise<envRefModel[]> {
    const response = await api.get(`/projects/${projectID}/envs/refs`)
    return response.data as envRefModel[]
}

export async function createEnv(projectID: string, model: envCreateModel): Promise<envModel> {
    const response = await api.post(`/projects/${projectID}/envs`, model)
    return response.data as envModel
}