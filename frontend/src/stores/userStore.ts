import { fetchUserResourceRefs, getUserInfo } from "@/api/user";
import type { appRefModel } from "@/types/app";
import type { envRefModel } from "@/types/env";
import type { projectRefModel } from "@/types/project";
import type { userModel, userResourcesModel } from "@/types/user";
import { defineStore } from "pinia";

export const useUserStore = defineStore('userStore', {
    state: () => ({
        user: null as userModel | null,
        userResources: null as userResourcesModel | null,
        activeProjectRef: null as projectRefModel | null,
        projectRefs: [] as projectRefModel[],
        activeEnvRef: null as envRefModel | null,
        envRefs: [] as envRefModel[],
        activeAppRef: null as appRefModel | null,
        appRefs: [] as appRefModel[],
    }),
    actions: {
        async initUser() {
            const userID = localStorage.getItem('userID');
            if (userID) {
                // Fetch user info from the server or cache
                const resp = await getUserInfo(userID);
                this.user = resp.data;
            } else {
                this.clearUser();
            }
        },
        async fetchUserResourceRefs() {
            // This method can be used to fetch user-related resource references
            // For now, it just initializes the user if not already done
            if (!this.user) {
                await this.initUser();
            }

            this.userResources = await fetchUserResourceRefs();
            this.projectRefs = this.userResources.projects;
            return this.userResources;
        },
        async activateApp(appID: string) {
            if (!this.userResources) {
                this.fetchUserResourceRefs();
            }

            this.activeAppRef = this.userResources.apps.find(app => app.appID === appID) || null;
            if (this.activeAppRef) {
                this.envRefs = this.userResources.envs.filter(env => env.projectID === this.activeAppRef.projectID);
                this.activeEnvRef = this.userResources.envs.find(env => env.envID === this.activeAppRef.envID) || null;
            }
            if (this.activeEnvRef) {
                this.activeProjectRef = this.projectRefs.find(project => project.projectID === this.activeEnvRef.projectID) || null;
            }
        },
        async activateEnv(envID: string) {
            if (!this.userResources) {
                this.fetchUserResourceRefs();
            }

            this.activeEnvRef = this.userResources.envs.find(env => env.envID === envID) || null;

            if (this.activeEnvRef) {
                this.activeProjectRef = this.projectRefs.find(project => project.projectID === this.activeEnvRef.projectID) || null;
                this.appRefs = this.userResources.apps.filter(app => app.envID === envID);
                if (this.activeAppRef && this.activeAppRef.envID !== this.activeEnvRef.envID) {
                    this.activeAppRef = this.appRefs.length > 0 ? this.appRefs[0] : null;
                }
            }
        },
        getUser() {
            return this.user;
        },
        setUser(newUser: userModel) {
            this.user = newUser;
            localStorage.setItem('userID', newUser.userID);
        },
        clearUser() {
            this.user = null;
            localStorage.removeItem('userID');
        },
    },
})