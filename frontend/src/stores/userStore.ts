import { listClusterNodeRefs } from "@/api/cluster";
import { fetchAdminResourceRefs, fetchUserResourceRefs, getUserInfo } from "@/api/user";
import type { appRefModel } from "@/types/app";
import type { clusterNodeRefModel, clusterRefModel } from "@/types/cluster";
import type { envRefModel } from "@/types/env";
import type { projectRefModel } from "@/types/project";
import type { adminResourcesModel, userModel, userResourcesModel } from "@/types/user";
import { defineStore } from "pinia";

export const useUserStore = defineStore('userStore', {
    state: () => ({
        user: null as userModel | null,

        // Admin resource references
        adminResources: null as adminResourcesModel | null,
        activeClusterRef: null as clusterRefModel | null,
        activeClusterNodeRef: null as clusterNodeRefModel | null,

        // User resource references
        userResources: null as userResourcesModel | null,
        activeProjectRef: null as projectRefModel | null,
        activeEnvRef: null as envRefModel | null,
        activeAppRef: null as appRefModel | null,
    }),
    actions: {
        async initUser() {
            const userID = localStorage.getItem('userID');
            if (userID) {
                const resp = await getUserInfo(userID);
                this.user = resp.data;
            } else {
                this.clearUser();
            }
        },
        setUser(newUser: userModel) {
            this.user = newUser;
            localStorage.setItem('userID', newUser.userID);
        },
        clearUser() {
            this.user = null;
            localStorage.removeItem('userID');
        },
        async fetchAdminResourceRefs() {
            if (!this.user) {
                await this.initUser();
                if (!this.user || this.user.role !== 'admin') {
                    return null;
                }
            }
            this.adminResources = await fetchAdminResourceRefs();

            return this.adminResources;
        },
        activateCluster(clusterID: string) {
            this.activeClusterRef = this.adminResources.clusters.find(cluster => cluster.clusterID === clusterID) || null;
        },
        activateClusterNode(clusterID: string, nodeName: string) {
            this.activateCluster(clusterID);

            if (this.adminResources?.clusterNodes) {
                this.activeClusterNodeRef = this.adminResources.clusterNodes.find((node: { clusterID: string; nodeName: string; }) => node.clusterID === clusterID && node.nodeName === nodeName) || null;
            }
        },
        async addOrUpdateCluster(cluster: clusterRefModel) {
            const index = this.adminResources.clusters.findIndex(e => e.clusterID === cluster.clusterID);
            if (index !== -1) {
                this.adminResources.clusters[index] = cluster;
            } else {
                this.adminResources.clusters.push(cluster);
                const nodeRefs = await listClusterNodeRefs(cluster.clusterID)
                this.adminResources.clusterNodes.push(...nodeRefs);
            }
        },
        deleteCluster(clusterID: string) {
            const index = this.adminResources.clusters.findIndex(e => e.clusterID === clusterID);
            if (index !== -1) {
                this.adminResources.clusters.splice(index, 1);
            }

            this.adminResources.clusterNodes = this.adminResources.clusterNodes.filter(node => node.clusterID !== clusterID);
            if (this.activeClusterRef && this.activeClusterRef.clusterID === clusterID) {
                this.activeClusterRef = null;
                this.activeClusterNodeRef = null;
            }
        },
        async fetchUserResourceRefs() {
            if (!this.user) {
                await this.initUser();
            }

            this.userResources = await fetchUserResourceRefs();
            return this.userResources;
        },
        activateApp(appID: string) {
            this.activeAppRef = this.userResources.apps.find(app => app.appID === appID) || null;
            if (this.activeAppRef) {
                localStorage.setItem('lastActiveAppID', this.activeAppRef.appID);
                this.activeEnvRef = this.userResources.envs.find(env => env.envID === this.activeAppRef.envID) || null;
            }
            if (this.activeEnvRef) {
                localStorage.setItem('lastActiveEnvID', this.activeEnvRef.envID);
                this.activeProjectRef = this.userResources.projects.find(project => project.projectID === this.activeEnvRef.projectID) || null;
            }
            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
            }
        },
        activateEnv(envID: string) {
            this.activeEnvRef = this.userResources.envs.find(env => env.envID === envID) || null;
            if (this.activeEnvRef) {
                localStorage.setItem('lastActiveEnvID', this.activeEnvRef.envID);
                this.activeProjectRef = this.userResources.projects.find(project => project.projectID === this.activeEnvRef.projectID) || null;
            }
            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
            }
        },
        ensureActiveProject() {
            if (this.activeProjectRef) {
                return;
            }

            if (!this.userResources.projects || this.userResources.projects.length === 0) {
                this.activeProjectRef = null;
                return;
            }

            const lastActiveProjectID = localStorage.getItem('lastActiveProjectID');
            if (lastActiveProjectID) {
                this.activeProjectRef = this.userResources.projects.find(project => project.projectID === lastActiveProjectID) || null;
            }

            if (!this.activeProjectRef) {
                this.activeProjectRef = this.userResources.projects.length > 0 ? this.userResources.projects[0] : null;
            }

            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
            }

            const lastActiveEnvID = localStorage.getItem('lastActiveEnvID');
            if (lastActiveEnvID) {
                const lastActiveEnv = this.userResources.envs.find(env => env.envID === lastActiveEnvID);
                if (lastActiveEnv && lastActiveEnv.projectID === this.activeProjectRef.projectID) {
                    this.activeEnvRef = lastActiveEnv;
                } else {
                    this.activeEnvRef = this.userResources.envs.find(env => env.projectID === this.activeProjectRef.projectID) || null;
                }
            }
            if (!this.activeEnvRef) {
                this.activeEnvRef = this.userResources.envs.find(env => env.projectID === this.activeProjectRef.projectID) || null;
            }
        },
        activateProject(projectID: string) {
            this.activeProjectRef = this.userResources.projects.find(project => project.projectID === projectID) || null;
            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
                this.activeEnvRef = this.userResources.envs.find(env => env.projectID === this.activeProjectRef.projectID) || null;
            }
        },
        addOrUpdateApp(app: appRefModel) {
            const index = this.userResources.apps.findIndex(a => a.appID === app.appID);
            if (index !== -1) {
                this.userResources.apps[index] = app;
            } else {
                this.userResources.apps.push(app);
            }
            if (this.activeAppRef && this.activeAppRef.appID === app.appID) {
                this.activeAppRef = app;
            }
        },
        deleteApp(appID: string) {
            const index = this.userResources.apps.findIndex(a => a.appID === appID);
            if (index !== -1) {
                this.userResources.apps.splice(index, 1);
            }
        },
        addOrUpdateEnv(env: envRefModel) {
            const index = this.userResources.envs.findIndex(e => e.envID === env.envID);
            if (index !== -1) {
                this.userResources.envs[index] = env;
            } else {
                this.userResources.envs.push(env);
            }
        },
        deleteEnv(envID: string) {
            const index = this.userResources.envs.findIndex(e => e.envID === envID);
            if (index !== -1) {
                this.userResources.envs.splice(index, 1);
            }
        },
        addOrUpdateProject(project: projectRefModel) {
            const index = this.userResources.projects.findIndex(p => p.projectID === project.projectID);
            if (index !== -1) {
                this.userResources.projects[index] = project;
            } else {
                this.userResources.projects.push(project);
            }
        },
        deleteProject(projectID: string) {
            const index = this.userResources.projects.findIndex(p => p.projectID === projectID);
            if (index !== -1) {
                this.userResources.projects.splice(index, 1);
            }
        },
    },
    getters: {
        getUser(state): userModel | null {
            return state.user;
        },
        getCurrentClusterRefs(state): clusterRefModel[] {
            return state.adminResources?.clusters ?? [];
        },
        getCurrentClusterNodeRefs(state): clusterNodeRefModel[] {
            if (!state.adminResources || !state.activeClusterRef) return [];
            return state.adminResources.clusterNodes.filter(node => node.clusterID === state.activeClusterRef.clusterID);
        },
        getCurrentAppRefs(state): appRefModel[] {
            if (!state.userResources || !state.activeEnvRef) return [];
            return state.userResources.apps.filter(app => app.envID === state.activeEnvRef.envID);
        },
        getCurrentEnvRefs(state): envRefModel[] {
            if (!state.userResources || !state.activeProjectRef) return [];
            return state.userResources.envs.filter(env => env.projectID === state.activeProjectRef.projectID);
        },
        getCurrentProjectRefs(state): projectRefModel[] {
            return state.userResources?.projects ?? [];
        }
    }
})
