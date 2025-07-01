<script setup lang="ts">
import { getApp, getAppRunningInfoUrl } from "@/api/app";
import Badge from "@/components/ui/badge/Badge.vue";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, useSidebar } from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useUserStore } from "@/stores/userStore";
import type { appModel, appRunningInfoModel } from "@/types/app";
import {
  Archive,
  Boxes,
  Dot,
  History,
  Monitor,
  PanelLeftClose,
  PanelLeftOpen,
  Settings2,
  SquarePen,
} from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { toast } from "vue-sonner";
import AppActions from "./AppActions.vue";
import Breadcrumb from "./breadcrumb/AppManagerBreadcrumb.vue";
import { appStatusDisplay } from "./data/appStatus";
import InstanceList from "./instance/InstanceList.vue";
import SettingDialog from "./setting/SettingDialog.vue";
import Settings from "./setting/Settings.vue";
import UpdateApp from "./UpdateApp.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();
const appID = route.params.id as string;

const userStore = useUserStore();
const { activeAppRef } = storeToRefs(userStore);
const currentTab = ref("overview");

const app = ref<appModel | null>(null);

async function fetchAppInfo(appID?: string) {
  if (appID) {
    app.value = await getApp(appID);
  }
}

const appRunningInfo = ref<appRunningInfoModel | null>(null);

let es: EventSource | null = null;
function fetchAppRunningInfo(appID?: string) {
  if (appID) {
    appRunningInfo.value = null; // Clear previous logs

    const appRunningInfoUrl = getAppRunningInfoUrl(appID);

    if (es) {
      es.close();
      es = null;
    }

    es = new EventSource(appRunningInfoUrl, { withCredentials: true });
    es.onmessage = (event) => {
      if (!event.data) {
        return;
      }
      appRunningInfo.value = JSON.parse(event.data);
    };
    es.onerror = (error) => {
      if (es) {
        es.close();
        es = null;
      }
      toast.dismiss();
      toast.error("应用组件运行信息获取失败", {
        description: "请检查网络连接或稍后重试。",
      });
    };
  }
}

onMounted(async () => {
  await fetchAppInfo(appID);
  fetchAppRunningInfo(appID);
});

watch(activeAppRef, async (newAppRef) => {
  if (newAppRef && newAppRef.appID !== app.value?.appID) {
    await fetchAppInfo(newAppRef.appID);
    fetchAppRunningInfo(newAppRef.appID);
  }
});

const monitorExtensionInstalled = ref(false);
const logsExtensionInstalled = ref(false);
const deployedInSourceCode = ref(false);

const settingDialogOpen = ref(false);
const openUpdateAppInfoDialog = ref(false);

const appStatus = computed(() => {
  return appStatusDisplay(appRunningInfo.value?.status || "unknown");
});
</script>

<template>
  <SidebarInset>
    <header
      class="flex h-12 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12"
    >
      <div class="flex items-center gap-2 px-4">
        <Button
          variant="ghost"
          @click="toggleSidebar"
          class="h-8 w-8 text-muted-foreground hover:text-primary"
        >
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
          <component
            :is="appStatus.icon"
            :class="`w-16 h-16 p-1 rounded-lg ${appStatus.class}`"
            stroke-width="1.2"
          />
          <div class="space-y-2 flex-1 gap-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <h1 class="text-xl font-semibold">
                  {{ app?.displayName || "应用名称" }}
                </h1>
                <SquarePen
                  class="w-4 h-4 text-muted-foreground hover:text-primary"
                  @click="openUpdateAppInfoDialog = true"
                />
                <Separator orientation="vertical" class="h-4" />
                <Badge
                  variant="secondary"
                  class="font-mono text-muted-foreground"
                >
                  类型：{{ app?.workloadType || "未知" }}</Badge
                >
                <Separator orientation="vertical" class="h-4" />
                <Badge
                  v-if="
                    app?.edition &&
                    appRunningInfo?.actualEdition &&
                    app.edition != appRunningInfo.actualEdition
                  "
                  variant="secondary"
                  class="font-mono text-blue-500 hover:text-blue-500 bg-blue-50 dark:bg-blue-950"
                >
                  <Dot
                    class="text-blue-500"
                    stroke-width="8"
                  />版本(有更新）：{{ appRunningInfo.actualEdition }}
                </Badge>
                <Badge
                  v-else
                  variant="secondary"
                  class="font-mono text-muted-foreground"
                  >版本：{{ appRunningInfo?.edition || "未知" }}</Badge
                >
              </div>

              <div style="margin-left: auto">
                <Button variant="secondary" size="sm" :class="appStatus.class">
                  <component :is="appStatus.icon" class="w-5 h-5 mr-1" />
                  <span>{{ appStatus.label }}</span>
                </Button>
              </div>
            </div>
            <div class="flex items-center justify-between">
              <p class="text-sm text-muted-foreground">
                {{ app?.description || "写一句话描述该应用吧。" }}
              </p>
              <div
                class="flex items-center gap-4 text-sm text-muted-foreground"
              >
                <AppActions
                  v-if="appRunningInfo"
                  :appRunningInfo="appRunningInfo"
                  :appEdition="app?.edition"
                  @action-completed="fetchAppInfo(appID)"
                />
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
          <Button
            variant="ghost"
            size="sm"
            class="flex ml-4"
            @click="settingDialogOpen = true"
          >
            <Settings2 class="w-4 h-4 mr-2" />
            设置
          </Button>
        </div>
        <Separator class="h-4" />
        <TabsContent value="overview">
          <InstanceList
            :appID="appID"
            :instances="appRunningInfo?.instances || []"
          />
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
    <UpdateApp
      v-model="openUpdateAppInfoDialog"
      :app="app"
      @app-updated="fetchAppInfo(app.appID)"
    />
  </SidebarInset>
</template>
