<script setup lang="ts">
import { getApp } from "@/api/app";
import Badge from "@/components/ui/badge/Badge.vue";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
    SidebarInset,
    useSidebar
} from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { appModel } from "@/types/app";
import { Archive, Boxes, History, Monitor, PanelLeftClose, PanelLeftOpen, Settings2 } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import NodeActions from "./NodeActions.vue";
import Breadcrumb from "./breadcrumb/AppManagerBreadcrumb.vue";
// import { appStatusDisplay } from "./data/appStatus";
import { useUserStore } from "@/stores/userStore";
import InstanceList from "./instance/InstanceList.vue";
import SettingDialog from "./setting/SettingDialog.vue";
import Settings from "./setting/Settings.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();
const appID = route.params.id as string;

const userStore = useUserStore()
const { activeAppRef } = storeToRefs(userStore);

const currentTab = ref("overview");

const app = ref<appModel | null>(null);

async function fetchNodeInfo(appID?: string) {
    if (appID) {
        app.value = await getApp(appID)
    }
}

onMounted(async () => {
    await fetchNodeInfo(appID);
});

watch(activeAppRef, async (newAppRef) => {
    if (newAppRef && newAppRef.appID !== app.value?.appID) {
        await fetchNodeInfo(newAppRef.appID);
    }
});

const monitorExtensionInstalled = ref(false);
const logsExtensionInstalled = ref(false);
const deployedInSourceCode = ref(false);

const settingDialogOpen = ref(false);

const appStatus = computed(() => {
    // return appStatusDisplay(app.value?.status || 'unknown')
});
</script>

<template>
    <SidebarInset>
        <header
            class="flex h-12 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
            <div class="flex items-center gap-2 px-4">
                <Button variant="ghost" @click="toggleSidebar" class="h-8 w-8 text-muted-foreground hover:text-primary">
                    <PanelLeftOpen v-if="!open" />
                    <PanelLeftClose v-else />
                </Button>
                <Separator orientation="vertical" class="mr-2 h-4" />
                <Breadcrumb :appID="app?.appID" />
            </div>
        </header>
        <div class="flex flex-col gap-4 mx-4 border-t pt-4">
            <div class="flex items-center justify-between">
                <div class="flex justify-between space-x-4 w-full items-center">
                    <component :is="appStatus.icon" :class="`w-12 h-12 ${appStatus.fgColor} rounded-sm`"
                        stroke-width="1" />
                    <div class="space-y-2 flex-1 gap-4">
                        <div class="flex items-center justify-between">
                            <div class="flex items-center gap-2">
                                <h1 class="text-xl font-semibold">
                                    {{ app?.displayName || "应用名称" }}
                                </h1>
                                <Separator orientation="vertical" class="h-4" />
                                <Badge variant="secondary" class="font-mono text-muted-foreground">
                                    应用类型：{{ app?.workloadType || '未知' }}</Badge>
                                <Separator orientation="vertical" class="h-4" />
                                <Badge variant="secondary" class="font-mono text-muted-foreground">部署版本：{{ app?.edition
                                    ||
                                    '未知'
                                    }}</Badge>
                            </div>

                            <div style="margin-left:auto;">
                                <Button variant="secondary" size="sm" :class="`${appStatus.fgColor}`">
                                    <component :is="appStatus.icon" class="w-5 h-5 mr-1" />
                                    <span>{{ appStatus.label }}</span>
                                </Button>
                            </div>
                        </div>
                        <div class="flex items-center justify-between">
                            <p class="text-sm text-muted-foreground">
                                {{ app?.description || "写一句话描述该应用吧。" }}
                            </p>
                            <div class="flex items-center gap-4 text-sm text-muted-foreground">
                                <NodeActions v-if="app" :app="app" @action-completed="fetchNodeInfo(appID)" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <Tabs v-model="currentTab" class="">
                <div class="flex items-center justify-between">
                    <TabsList class="grid grid-cols-5">
                        <TabsTrigger value="overview">
                            <Boxes />
                            实例
                        </TabsTrigger>
                        <TabsTrigger value="monitor" :disabled="monitorExtensionInstalled">
                            <Monitor />
                            监控
                        </TabsTrigger>
                        <TabsTrigger value="logs" :disabled="logsExtensionInstalled">
                            <Archive />
                            归档日志
                        </TabsTrigger>
                        <TabsTrigger value="builds" :disabled="deployedInSourceCode">
                            <History />
                            构建历史
                        </TabsTrigger>
                        <TabsTrigger value="settings" :disabled="deployedInSourceCode">
                            <Settings2 />
                            设置
                        </TabsTrigger>
                    </TabsList>
                    <Button variant="default" class=" flex ml-4" @click="settingDialogOpen = true">
                        <Settings2 class="w-4 h-4 mr-2" />
                        设置
                    </Button>
                </div>
                <Separator class="h-4" />
                <TabsContent value="overview">
                    <InstanceList />
                </TabsContent>
                <TabsContent value="monitor">
                    <div class="mt-4 text-muted-foreground">监控功能开发中...</div>
                </TabsContent>
                <TabsContent value="logs">
                    <div class="mt-4 text-muted-foreground">日志功能开发中...</div>
                </TabsContent>
                <TabsContent value="builds">
                    <div class="mt-4 text-muted-foreground">构建历史功能开发中...</div>
                </TabsContent>
                <TabsContent value="settings">
                    <Settings v-if="app" :app="app" />
                </TabsContent>
            </Tabs>
        </div>
        <SettingDialog v-model="settingDialogOpen" />
    </SidebarInset>
</template>
