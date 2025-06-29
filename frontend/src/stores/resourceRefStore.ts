import { getAppRef } from '@/api/app';
import { fetchAppRefs, getEnvRef } from '@/api/env';
import { fetchEnvRefs, fetchProjectRefs, getProjectRef } from '@/api/project';
import type { appRefModel } from '@/types/app';
import type { envRefModel } from '@/types/env';
import type { projectRefModel } from '@/types/project';
import { defineStore } from 'pinia';

export const useResourceRefStore = defineStore('resourceRefStore', {
    state: () => ({
        activeProjectRef: null as projectRefModel | null,
        activeEnvRef: null as envRefModel | null,
        activeAppRef: null as appRefModel | null,

        projectRefs: [] as projectRefModel[],
        envRefs: [] as envRefModel[],
        appRefs: [] as appRefModel[],
        // clusterRefs: [] as clusterRefModel[],
    }),
    actions: {
        getActiveProjectRef() {
            return this.activeProjectRef;
        },
        async initFromAppID(appID: string) {
            this.activeAppRef = await getAppRef(appID)
            this.activeEnvRef = await getEnvRef(this.activeAppRef.envID)
            this.activeProjectRef = await getProjectRef(this.activeAppRef.projectID)
            this.projectRefs = await fetchProjectRefs()
            this.envRefs = await fetchEnvRefs(this.activeProjectRef.projectID)
            this.appRefs = await fetchAppRefs(this.activeEnvRef.envID)
            localStorage.setItem('lastActiveAppID', appID)
            localStorage.setItem('lastActiveEnvID', this.activeEnvRef.envID)
            localStorage.setItem('lastActiveProjectID', this.activeProjectRef.projectID);
        },
        async initFromEnvID(envID: string) {
            this.activeEnvRef = await getEnvRef(envID)
            this.envRefs = await fetchEnvRefs(this.activeEnvRef.projectID)
            this.activeProjectRef = await getProjectRef(this.activeEnvRef.projectID)
            this.projectRefs = await fetchProjectRefs()
            this.appRefs = await fetchAppRefs(envID)

            localStorage.setItem('lastActiveEnvID', envID)
            localStorage.setItem('lastActiveProjectID', this.activeEnvRef.projectID)
        },
        async initFromProjectID(projectID: string) {
            this.activeProjectRef = await getProjectRef(projectID)
            this.projectRefs = await fetchProjectRefs()
            this.envRefs = await fetchEnvRefs(projectID)
            if (this.activeEnvRef) {
                this.activeEnvRef = this.envRefs.length > 0 ? this.envRefs.find(env => env.envID === this.activeEnvRef.envID) || this.envRefs[0] : null;
            } else {
                this.activeEnvRef = this.envRefs.length > 0 ? this.envRefs[0] : null;
            }
            localStorage.setItem('lastActiveProjectID', projectID)
            localStorage.setItem('lastActiveEnvID', this.activeEnvRef?.envID || '')
        },
        async initFromProject() {
            this.projectRefs = await fetchProjectRefs()
            let lastActiveProjectID = localStorage.getItem('lastActiveProjectID');
            if (lastActiveProjectID) {
                const matched = this.projectRefs.find(project => project.projectID === lastActiveProjectID);
                if (matched) {
                    this.activeProjectRef = matched;
                }
            }
            if (!this.activeProjectRef && this.projectRefs.length > 0) {
                this.activeProjectRef = this.projectRefs[0];
            }

            if (this.activeProjectRef) {
                this.envRefs = await fetchEnvRefs(this.activeProjectRef.projectID)
                let lastActiveEnvID = localStorage.getItem('lastActiveEnvID');
                if (lastActiveEnvID) {
                    const matched = this.envRefs.find(env => env.envID === lastActiveEnvID);
                    if (matched) {
                        this.activeEnvRef = matched;
                    }
                }
                if (!this.activeEnvRef && this.envRefs.length > 0) {
                    this.activeEnvRef = this.envRefs[0];
                }
            }

            localStorage.setItem('lastActiveProjectID', this.activeProjectRef?.projectID || '')
            localStorage.setItem('lastActiveEnvID', this.activeEnvRef?.envID || '')
        },
        async setActiveAppRef(appRef: appRefModel) {
            this.activeAppRef = appRef;
            const appRefs = await fetchAppRefs(appRef.envID);
            if (appRefs) {
                this.setAppRefs(appRefs);
            }
        },
        setAppRefs(appRefs: appRefModel[]) {
            this.appRefs = appRefs;
        },
        clearEnvRefs() {
            this.activeEnvRef = null;
            this.envRefs = [];
            localStorage.removeItem('lastActiveEnvID');

            this.clearAppRefs();
        },
        clearAppRefs() {
            this.activeAppRef = null;
            this.appRefs = [];
            localStorage.removeItem('lastActiveAppID');
        },
        async switchProject(projectID: string) {
            this.clearEnvRefs()

            this.activeProjectRef = await getProjectRef(projectID)
            this.envRefs = await fetchEnvRefs(projectID)
            const lastActiveEnvID = localStorage.getItem('lastActiveEnvID');
            if (lastActiveEnvID) {
                const matched = this.envRefs.find(env => env.envID === lastActiveEnvID);
                if (matched) {
                    this.activeEnvRef = matched;
                }
            }
            if (!this.activeEnvRef && this.envRefs.length > 0) {
                this.activeEnvRef = this.envRefs[0]
            }

            localStorage.setItem('lastActiveProjectID', projectID)
            localStorage.setItem('lastActiveEnvID', this.activeEnvRef?.envID || '')
        },
        async switchEnv(envID: string) {
            // this.clearAppRefs()
            // this.activeEnvRef = await getEnvRef(envID)

            // if (this.activeProjectRef?.projectID !== this.activeEnvRef?.projectID) {
            //     this.activeProjectRef = await getProjectRef(this.activeEnvRef.projectID)
            // }

            // this.appRefs = await fetchAppRefs(envID)
            // this.activeAppRef = null

            // localStorage.setItem('lastActiveEnvID', envID)
            // localStorage.setItem('lastActiveProjectID', this.activeEnvRef?.projectID || '')
            this.initFromEnvID(envID);
        },
        async addEnv(newEnv: envRefModel) {
            this.envRefs.push(newEnv)

            if (!this.activeEnvRef) {
                this.activeEnvRef = newEnv
            }
        },
        async removeEnv(envID: string) {
            this.envRefs = this.envRefs.filter(env => env.envID !== envID)
            if (this.activeEnvRef?.envID === envID) {
                this.activeEnvRef = this.envRefs.length > 0 ? this.envRefs[0] : null;
            }
        },
        async switchApp(appID: string) {
            // this.activeAppRef = await getAppRef(appID)
            // if (this.activeProjectRef?.projectID !== this.activeAppRef?.projectID) {
            //     this.activeProjectRef = await getProjectRef(this.activeAppRef.projectID)
            // }
            // if (this.activeEnvRef?.envID !== this.activeAppRef?.envID) {
            //     this.activeEnvRef = await getEnvRef(this.activeAppRef.envID)
            // }

            // localStorage.setItem('lastActiveAppID', appID)
            // localStorage.setItem('lastActiveEnvID', this.activeAppRef?.envID || '')
            // localStorage.setItem('lastActiveProjectID', this.activeAppRef?.projectID || '')

            this.initFromAppID(appID)
        },
        async addApp(newApp: appRefModel) {
            this.appRefs.push(newApp)

            if (!this.activeAppRef) {
                this.activeAppRef = newApp
            }
        },
    },
})
