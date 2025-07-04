<script setup lang="ts">
import { getClusterNode } from "@/api/cluster";
import Badge from "@/components/ui/badge/Badge.vue";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
    SidebarInset,
    useSidebar
} from "@/components/ui/sidebar";
import { useUserStore } from "@/stores/userStore";
import type { clusterNodeModel } from "@/types/cluster";
import { CircleCheck, CircleDashed, PanelLeftClose, PanelLeftOpen, Server } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import Breadcrumb from "../breadcrumb/ClusterManagerBreadcrumb.vue";
import NodeActions from "./NodeActions.vue";

const { toggleSidebar, open } = useSidebar();

const router = useRouter();

const route = useRoute();
const clusterID = route.params.id as string;
const nodeName = route.params.nodeName as string;

const userStore = useUserStore()
const { activeClusterNodeRef } = storeToRefs(userStore);


const currentTab = ref("overview");

const node = ref<clusterNodeModel | null>(null);

async function fetchNodeInfo(clusterID?: string, nodeName?: string) {
    if (clusterID && nodeName) {
        node.value = await getClusterNode(clusterID, nodeName)
    }
}

onMounted(async () => {
    await fetchNodeInfo(clusterID, nodeName);
});

watch(activeClusterNodeRef, async (newClusterNode) => {
    if (newClusterNode && newClusterNode.nodeName !== node.value?.nodeName) {
        await fetchNodeInfo(clusterID, newClusterNode.nodeName);
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
                <Breadcrumb :clusterID="clusterID" :nodeName="nodeName" />
            </div>
        </header>
        <div class="flex flex-col gap-4 mx-4 border-t pt-4">
            <div class="flex items-center justify-between">
                <div class="flex justify-between space-x-4 w-full items-center">
                    <Server
                        :class="`${node?.ready ? 'w-16 h-16 p-1 rounded-lg text-green-500 bg-green-50' : 'text-gray-500'} rounded-sm`"
                        stroke-width="1" />
                    <div class="space-y-2 flex-1 gap-4">
                        <div class="flex items-center justify-between">
                            <div class="flex items-center gap-2">
                                <h1 class="text-xl font-semibold">
                                    {{ node?.nodeName || "节点名称" }}
                                </h1>
                                <Separator orientation="vertical" class="h-4" />
                                <Badge variant="secondary" class="font-mono text-muted-foreground">
                                    内网 IP：{{ node?.internalIP || '未知' }}</Badge>
                                <Separator orientation="vertical" class="h-4" />
                                <Badge variant="secondary" class="font-mono text-muted-foreground">版本：{{
                                    node?.kubeletVersion
                                    ||
                                    '未知'
                                }}</Badge>
                            </div>

                            <div style="margin-left:auto;">
                                <Button variant="secondary" size="sm"
                                    :class="`${node?.ready ? 'text-green-500 bg-green-500/10 hover:bg-green-500/10' : 'text-gray-500 bg-gray-500/10 hover:bg-gray-500/10'}`">
                                    <component :is="node?.ready ? CircleCheck : CircleDashed" class="w-4 h-4 mr-1" />
                                    <span>{{ node?.ready ? "就绪" : "未就绪" }}</span>
                                </Button>
                            </div>
                        </div>
                        <div class="flex items-center justify-between">
                            <p class="text-sm text-muted-foreground">
                                {{ node?.internalIP }}
                            </p>
                            <div class="flex items-center gap-4 text-sm text-muted-foreground">
                                <NodeActions v-if="node" :app="node"
                                    @action-completed="fetchNodeInfo(clusterID, node.nodeName)" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="mt-4 text-muted-foreground">正在路上...</div>
        </div>
    </SidebarInset>
</template>
