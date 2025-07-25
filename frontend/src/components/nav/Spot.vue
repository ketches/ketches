<script setup lang="ts">
import { useUserStore } from "@/stores/userStore.ts";
import { useMagicKeys, whenever } from "@vueuse/core";
import { GalleryHorizontalEnd, Grid2X2, Package, Search } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import {
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxRoot,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DialogTrigger,
  VisuallyHidden,
} from "reka-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import ScrollArea from "../ui/scroll-area/ScrollArea.vue";
import SidebarMenuButton from "../ui/sidebar/SidebarMenuButton.vue";

const router = useRouter();

const open = ref(false);
const userStore = useUserStore();
const { userResources, adminResources, user } = storeToRefs(userStore);

const { meta_k, ctrl_k } = useMagicKeys();
const anyKPressed = computed(() => meta_k.value || ctrl_k.value);
whenever(anyKPressed, () => {
  open.value = true;
});

function handleSelect(ev: CustomEvent, resourceType: string, resourceID?: string) {
  ev.preventDefault();
  open.value = false;

  switch (resourceType) {
    case "cluster":
      userStore.activateCluster(resourceID);
      router.push({ name: 'cluster-page', params: { id: resourceID } });
      break;
    case "clusterNode":
      const [clusterID, nodeName] = resourceID.split('/');
      userStore.activateClusterNode(clusterID, nodeName);
      router.push({ name: 'cluster-node-page', params: { id: clusterID, nodeName } });
      break;
    case "project":
      userStore.activateProject(resourceID);
      break;
    case "env":
      userStore.activateEnv(resourceID);
      router.push({ name: 'env-page', params: { id: resourceID } });
      break;
    case "app":
      userStore.activateApp(resourceID);
      router.push({ name: 'app-page', params: { id: resourceID } });
      break;
    default:
      console.warn(`Unknown resource type: ${resourceType}`);
  }
}
</script>

<template>
  <DialogRoot v-model:open="open" :returnFocusOnClose="false">
    <DialogTrigger class="w-full">
      <SidebarMenuButton class="dark:text-white border flex items-center justify-between text-muted-foreground">
        <Search />
        <span class="justify-center flex-1">聚焦</span>
        <span
          class="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded bg-muted px-1.5 font-mono text-[12px] font-medium text-muted-foreground/50 opacity-100">
          <kbd class="inline-flex items-center gap-0.5 align-middle">
            <span style="font-size:1.2em;line-height:1;display:inline-block;vertical-align:middle;">⌘</span>
            <span style="margin-left:2px;vertical-align:middle;">K, Ctrl K</span>
          </kbd>
        </span>
      </SidebarMenuButton>
    </DialogTrigger>

    <DialogPortal>
      <DialogOverlay class="bg-background/80 fixed inset-0 z-30" />
      <DialogContent
        class="fixed top-[15%] left-[50%] max-h-[85vh] w-[120vw] max-w-[36rem] translate-x-[-50%] text-sm rounded-xl overflow-hidden border border-muted-foreground/30 bg-card focus:outline-none z-[100] shadow-accent-foreground/25 shadow-2xl">
        <VisuallyHidden>
          <DialogTitle>Command Menu</DialogTitle>
          <DialogDescription>Search for command</DialogDescription>
        </VisuallyHidden>

        <ComboboxRoot :open="true">
          <ComboboxInput placeholder="搜索我的资源 ..."
            class="bg-transparent w-full px-4 py-3 outline-none placeholder-muted-foreground" @keydown.enter.prevent />
          <ComboboxContent v-if="user.role === 'admin' && adminResources"
            class="border-t border-muted-foreground/30 max-h-[20rem] overflow-y-auto" @escape-key-down="open = false">
            <ComboboxEmpty class="text-center text-muted-foreground p-4">
              No results
            </ComboboxEmpty>
            <ScrollArea class="w-full h-full flex-1 min-h-0 flex flex-col">
              <ComboboxGroup v-if="adminResources.clusters.length > 0" class="px-4 pb-2">
                <ComboboxLabel
                  class="inline-flex w-full items-center gap-4 text-muted-foreground/70 font-semibold mt-3">
                  <Package class="h-4 w-4" />
                  集群
                </ComboboxLabel>
                <ComboboxItem v-for="item in adminResources.clusters" :key="item.clusterID" :value="item"
                  class="cursor-default pl-8 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'cluster', item.clusterID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
              </ComboboxGroup>
              <ComboboxGroup v-if="adminResources.clusters.length > 0" class="px-4 pb-2">
                <ComboboxLabel
                  class="inline-flex w-full items-center gap-4 text-muted-foreground/70 font-semibold mt-3">
                  <Package class="h-4 w-4" />
                  节点
                </ComboboxLabel>
                <ComboboxItem v-for="item in adminResources.clusterNodes" :key="item.nodeName" :value="item"
                  class="cursor-default pl-8 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'clusterNode', item.clusterDisplayName + '/' + item.nodeName)">
                  <span>{{ item.clusterDisplayName }}/{{ item.nodeName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.nodeIP }}</span>
                </ComboboxItem>
              </ComboboxGroup>
            </ScrollArea>
          </ComboboxContent>
          <ComboboxContent v-else-if="user.role === 'user' && userResources"
            class="border-t border-muted-foreground/30 max-h-[20rem] overflow-y-auto" @escape-key-down="open = false">
            <ComboboxEmpty class="text-center text-muted-foreground p-4">
              No results
            </ComboboxEmpty>
            <ScrollArea class="w-full h-full flex-1 min-h-0 flex flex-col">

              <ComboboxGroup v-if="userResources.apps.length > 0" class="px-4 pb-2">
                <ComboboxLabel
                  class="inline-flex w-full items-center gap-4 text-muted-foreground/70 font-semibold mt-3">
                  <Package class="h-4 w-4" />
                  应用
                </ComboboxLabel>
                <ComboboxItem v-for="item in userResources.apps" :key="item.appID" :value="item"
                  class="cursor-default pl-8 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'app', item.appID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
              </ComboboxGroup>
              <ComboboxGroup v-if="userResources.envs.length > 0" class="px-4 pb-2">
                <ComboboxLabel
                  class="inline-flex w-full items-center gap-4 text-muted-foreground/70 font-semibold mt-3">
                  <Grid2X2 class="h-4 w-4" />
                  环境
                </ComboboxLabel>
                <ComboboxItem v-for="item in userResources.envs" :key="item.envID" :value="item"
                  class="cursor-default pl-8 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'env', item.envID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
              </ComboboxGroup>
              <ComboboxGroup v-if="userResources.projects.length > 0" class="px-4 pb-2">
                <ComboboxLabel
                  class="inline-flex w-full items-center gap-4 text-muted-foreground/70 font-semibold mt-3">
                  <GalleryHorizontalEnd class="h-4 w-4" />
                  项目
                </ComboboxLabel>
                <ComboboxItem v-for="item in userResources.projects" :key="item.projectID" :value="item"
                  class="cursor-default pl-8 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'project', item.projectID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
              </ComboboxGroup>
            </ScrollArea>
          </ComboboxContent>
        </ComboboxRoot>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
