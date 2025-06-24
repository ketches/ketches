import api from '@/api/axios';
import type { QueryAndPagedRequest } from '@/types/common';
import type { createProjectModel, projectMemberModel, projectModel, projectRefModel } from '@/types/project';
import type { userRefModel } from '@/types/user';

// export const useProjectStore = defineStore('projectStore', {
//     state: () => ({
//         projects: [] as projectModel[],
//         activeProject: null as projectModel | null,
//     }),
//     actions: {
//         setProjects(newProjects: projectModel[]) {
//             this.projects = newProjects;
//             const lastActiveProjectID = localStorage.getItem('lastActiveProjectID');
//             if (lastActiveProjectID) {
//                 const match = newProjects.find(p => p.projectID === lastActiveProjectID);
//                 if (match) {
//                     this.setActiveProject(match);
//                     return
//                 }
//             }

//             if (newProjects.length > 0) {
//                 this.setActiveProject(newProjects[0]);
//             } else {
//                 this.clearActiveProject();
//             }
//         },
//         addProject(newProject: projectModel) {
//             if (this.projects.length == 0) {
//                 this.setActiveProject(newProject);
//             }
//             this.projects.push(newProject);
//         },
//         removeProject(projectID: string) {
//             this.projects = this.projects.filter(project => project.projectID !== projectID);
//             if (this.activeProject && this.activeProject.projectID === projectID) {
//                 this.clearActiveProject();
//                 if (this.projects.length > 0) {
//                     this.setActiveProject(this.projects[0]);
//                 }
//             }
//         },
//         setActiveProject(project: projectModel | null) {
//             this.activeProject = project;
//             localStorage.setItem('lastActiveProjectID', project ? project.projectID : '');
//         },
//         clearActiveProject() {
//             this.activeProject = null;
//             localStorage.removeItem('lastActiveProjectID');
//         },
//     },
// })

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
