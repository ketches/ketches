import { fetchClusterRefs } from "@/api/cluster";
import type { clusterRefModel } from "@/types/cluster";
import { defineStore } from "pinia";

export const useClusterStore = defineStore('clusterStore', {
    state: () => ({
        clusterRefs: [] as clusterRefModel[],
        activeClusterRef: null as clusterRefModel | null,
    }),
    actions: {
        async loadClusterRefs(refresh = false) {
            if (this.clusterRefs.length === 0 || refresh) {
                this.clusterRefs = await fetchClusterRefs()
            }
        },
        setClusterRefs(newClusterRefs: clusterRefModel[]) {
            this.clusterRefs = newClusterRefs;
        },
        addClusterRef(newClusterRef: clusterRefModel) {
            this.clusterRefs.push(newClusterRef);
        },
        removeClusterRef(clusterID: string) {
            this.clusterRefs = this.clusterRefs.filter(cluster => cluster.clusterID !== clusterID);
        },
        setActiveClusterRef(clusterRef: clusterRefModel) {
            this.activeClusterRef = clusterRef;
        },
    },
});