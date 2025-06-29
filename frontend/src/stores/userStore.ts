import { fetchUserResourceRefs, getUserInfo } from "@/api/user";
import type { appRefModel } from "@/types/app";
import type { clusterRefModel } from "@/types/cluster";
import type { envRefModel } from "@/types/env";
import type { projectRefModel } from "@/types/project";
import type { userModel, userResourcesModel } from "@/types/user";
import { defineStore } from "pinia";

export const useUserStore = defineStore('userStore', {
    state: () => ({
        user: null as userModel | null,

        // Admin resource references
        clusterRefs: [] as clusterRefModel[], // Assuming clusterRefModel has clusterID and name

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
        // getter 方式已在下方实现
        setUser(newUser: userModel) {
            this.user = newUser;
            localStorage.setItem('userID', newUser.userID);
        },
        clearUser() {
            this.user = null;
            localStorage.removeItem('userID');
        },
        async fetchUserResourceRefs() {
            if (!this.user) {
                await this.initUser();
            }

            this.userResources = await fetchUserResourceRefs();
            return this.userResources;
        },
        // async activateApp(appID: string) {
        activateApp(appID: string) {
            // if (!this.userResources) {
            //     await this.fetchUserResourceRefs();
            // }
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
        //  activateEnv(envID: string) {
        activateEnv(envID: string) {
            //     if (!this.userResources) {
            //         await this.fetchUserResourceRefs();
            //     }
            this.activeEnvRef = this.userResources.envs.find(env => env.envID === envID) || null;
            if (this.activeEnvRef) {
                localStorage.setItem('lastActiveEnvID', this.activeEnvRef.envID);
                this.activeProjectRef = this.userResources.projects.find(project => project.projectID === this.activeEnvRef.projectID) || null;
            }
            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
            }
        },
        // async ensureActiveProject() {
        ensureActiveProject() {
            if (this.activeProjectRef) {
                return;
            }

            // if (!this.userResources) {
            //     await this.fetchUserResourceRefs();
            // }

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
        },
        // async activateProject(projectID: string) {
        activateProject(projectID: string) {
            // if (!this.userResources) {
            //     await this.fetchUserResourceRefs();
            // }
            this.activeProjectRef = this.userResources.projects.find(project => project.projectID === projectID) || null;
            if (this.activeProjectRef) {
                localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
                this.activeEnvRef = this.userResources.envs.find(env => env.projectID === this.activeProjectRef.projectID) || null;
            }
        },
        // getter 方式已在下方实现
        // async addOrUpdateApp(app: appRefModel) {
        addOrUpdateApp(app: appRefModel) {
            // if (!this.userResources) {
            //     await this.fetchUserResourceRefs();
            // }

            const index = this.userResources.apps.findIndex(a => a.appID === app.appID);
            if (index !== -1) {
                this.userResources.apps[index] = app;
            } else {
                this.userResources.apps.push(app);
            }
        },
        // async deleteApp(appID: string) {
        deleteApp(appID: string) {
            // if (!this.userResources) {
            //     await this.fetchUserResourceRefs();
            // }

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
        getCurrentAppRefs(state): appRefModel[] {
            if (!state.userResources || !state.activeAppRef) return [];
            return state.userResources.apps.filter(app => app.envID === state.activeAppRef.envID);
        },
        getCurrentEnvRefs(state): envRefModel[] {
            if (!state.userResources || !state.activeEnvRef) return [];
            return state.userResources.envs.filter(env => env.projectID === state.activeEnvRef.projectID);
        },
        getCurrentProjectRefs(state): projectRefModel[] {
            return state.userResources?.projects ?? [];
        }
    }
})
