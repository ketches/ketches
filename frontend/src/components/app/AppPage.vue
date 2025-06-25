<script setup lang="ts">
import { appStatusToText, getApp } from "@/api/app";
import Badge from "@/components/ui/badge/Badge.vue";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  useSidebar
} from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useResourceRefStore } from "@/stores/resourceRefStore";
import type { appModel } from "@/types/app";
import { Archive, Boxes, History, Monitor, PackageCheck, PackageMinus, PanelLeftClose, PanelLeftOpen, Settings2, Undo2 } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import AppActions from "./AppActions.vue";
import Breadcrumb from "./breadcrumb/AppManagerBreadcrumb.vue";
import InstanceList from "./instance/InstanceList.vue";
import SettingDialog from "./setting/SettingDialog.vue";
import Settings from "./setting/Settings.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();
const appID = route.params.id as string;

const resourceRefStore = useResourceRefStore()
const { activeAppRef } = storeToRefs(resourceRefStore);
const currentTab = ref("overview");

const app = ref<appModel | null>(null);

async function fetchAppInfo(appID?: string) {
  if (appID) {
    app.value = await getApp(appID)
  }
}

onMounted(async () => {
  await fetchAppInfo(appID);
});

watch(activeAppRef, async (newAppRef) => {
  if (newAppRef && newAppRef.appID !== app.value?.appID) {
    await fetchAppInfo(newAppRef.appID);
  }
});

const statusColor = (status: string) => {
  switch (status) {
    case "running":
      return "text-green-500";
    case "starting":
    case "rollingUpdate":
      return "text-blue-500";
    case "stopping":
      return "text-yellow-500";
    default:
      return "text-gray-500";
  }
};

const monitorExtensionInstalled = ref(false);
const logsExtensionInstalled = ref(false);
const deployedInSourceCode = ref(false);

const settingDialogOpen = ref(false);
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
        <div class="flex items-center gap-4">
          <h1 class="text-2xl font-bold">{{ app?.displayName || "应用名称" }}</h1>
          <Badge variant="secondary" :class="statusColor(app?.status || 'unknown')">
            <PackageCheck v-if="app?.deployed === true" class="w-4 h-4 mr-1" />
            <PackageMinus v-else class="w-4 h-4 mr-1" />
            {{ appStatusToText(app?.status || "unknown") }}
          </Badge>
        </div>
        <AppActions v-if="app" :app="app" @action-completed="fetchAppInfo(appID)" />
      </div>
      <div class="flex items-center gap-4 text-sm text-muted-foreground">
        <Button variant="link" class="text-sm font-normal text-muted-foreground hover:text-primary p-0 h-auto" size="sm"
          @click="$router.push({ name: 'app' })">
          <Undo2 class="w-4 h-4 mr-1" /> 返回应用列表
        </Button>
        <Separator orientation="vertical" class="h-4" />
        <div class="font-mono">{{ app?.slug }}</div>
        <Separator orientation="vertical" class="h-4" />
        <span>应用类型: <Badge variant="secondary">{{ app?.workloadType || '未知' }}</Badge></span>
        <Separator orientation="vertical" class="h-4" />
        <span>部署版本: <Badge variant="secondary" class="font-mono">{{ app?.deployVersion || '未知' }}</Badge></span>
      </div>

      <Tabs v-model="currentTab" class="mt-2">
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
