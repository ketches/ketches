<script setup lang="ts">
import { getCluster } from "@/api/cluster";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
    SidebarInset,
    useSidebar
} from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useUserStore } from "@/stores/userStore";
import type { clusterModel } from "@/types/cluster";
import { Blocks, Boxes, PanelLeftClose, PanelLeftOpen } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import Breadcrumb from "./breadcrumb/ClusterManagerBreadcrumb.vue";
import ClusterActions from "./ClusterActions.vue";
import ExtensionList from "./ExtensionList.vue";
import NodeList from "./node/NodeList.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();
const clusterID = route.params.id as string;

const userStore = useUserStore()
const { activeClusterRef, adminResources } = storeToRefs(userStore);
const currentTab = ref("nodes");

const cluster = ref<clusterModel | null>(null);

async function fetchClusterInfo(clusterID?: string) {
    if (clusterID) {
        cluster.value = await getCluster(clusterID)
    }
}

onMounted(async () => {
    await fetchClusterInfo(clusterID);
});

watch(activeClusterRef, async (newClusterRef) => {
    if (newClusterRef && newClusterRef.clusterID !== cluster.value?.clusterID) {
        await fetchClusterInfo(newClusterRef.clusterID);
    }
});

const monitorExtensionInstalled = ref(false);

const settingDialogOpen = ref(false);

</script>

<template>
    <SidebarInset>
        <header
            class="flex h-12 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
            <div class="flex items-center px-4">
                <Button variant="ghost" @click="toggleSidebar" class="h-8 w-8 text-muted-foreground hover:text-primary">
                    <PanelLeftOpen v-if="!open" />
                    <PanelLeftClose v-else />
                </Button>
                <Separator orientation="vertical" class="mr-2 h-4" />
                <Breadcrumb :clusterID="cluster?.clusterID" :clusterRefs="adminResources?.clusters" />
            </div>
        </header>
        <div class="flex flex-col gap-4 mx-4 border-t pt-4">
            <div class="flex items-center justify-between">
                <div class="flex justify-between space-x-4 w-full items-center">
                    <Boxes :class="`w-12 h-12 rounded-sm`" stroke-width="1" />
                    <div class="space-y-2 flex-1 gap-4">
                        <div class="flex items-center justify-between">
                            <div class="flex items-center gap-2">
                                <h1 class="text-xl font-semibold">
                                    {{ cluster?.displayName || "集群名称" }}
                                </h1>
                            </div>
                        </div>
                        <div class="flex items-center justify-between">
                            <p class="text-sm text-muted-foreground">
                                {{ cluster?.description || "写一句话描述该集群吧。" }}
                            </p>
                            <div class="flex items-center gap-4 text-sm text-muted-foreground">
                                <ClusterActions v-if="cluster" :cluster="cluster"
                                    @action-completed="fetchClusterInfo(clusterID)" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <Tabs v-model="currentTab" class="">
                <div class="flex items-center justify-between">
                    <TabsList class="grid grid-cols-2">
                        <TabsTrigger value="nodes">
                            <Boxes />
                            节点
                        </TabsTrigger>
                        <TabsTrigger value="extensions">
                            <Blocks />
                            扩展
                        </TabsTrigger>
                    </TabsList>
                </div>
                <Separator class="h-4" />
                <TabsContent value="nodes">
                    <NodeList v-if="cluster" :cluster="cluster" />
                </TabsContent>
                <TabsContent value="extensions" class="h-full">
                    <ExtensionList :cluster="cluster" />
                </TabsContent>
            </Tabs>
        </div>
    </SidebarInset>
</template>
