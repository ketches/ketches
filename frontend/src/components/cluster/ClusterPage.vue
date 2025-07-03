<script setup lang="ts">
import { getCluster } from "@/api/cluster";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
    SidebarInset,
    useSidebar
} from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useClusterStore } from "@/stores/clusterStore";
import type { clusterModel } from "@/types/cluster";
import { Boxes, Monitor, PanelLeftClose, PanelLeftOpen, Settings2 } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import Breadcrumb from "./breadcrumb/ClusterManagerBreadcrumb.vue";
import ClusterActions from "./ClusterActions.vue";
import NodeList from "./node/NodeList.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();
const clusterID = route.params.id as string;

const clusterStore = useClusterStore()
const { activeClusterRef, clusterRefs } = storeToRefs(clusterStore);
const currentTab = ref("overview");

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
const logsExtensionInstalled = ref(false);
const deployedInSourceCode = ref(false);

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
                <Breadcrumb :clusterID="cluster?.clusterID" :clusterRefs="clusterRefs" />
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
                        <TabsTrigger value="overview">
                            <Boxes />
                            节点
                        </TabsTrigger>
                        <TabsTrigger value="monitor" :disabled="monitorExtensionInstalled">
                            <Monitor />
                            监控
                        </TabsTrigger>

                    </TabsList>
                    <Button variant="default" class=" flex ml-4" @click="settingDialogOpen = true">
                        <Settings2 class="w-4 h-4 mr-2" />
                        设置
                    </Button>
                </div>
                <Separator class="h-4" />
                <TabsContent value="overview">
                    <NodeList />
                </TabsContent>
                <TabsContent value="monitor">
                    <div class="mt-4 text-muted-foreground">监控功能开发中...</div>
                </TabsContent>
            </Tabs>
        </div>
    </SidebarInset>
</template>
