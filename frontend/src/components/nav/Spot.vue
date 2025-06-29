<script setup lang="ts">
import { useResourceRefStore } from "@/stores/resourceRefStore.ts";
import { useUserStore } from "@/stores/userStore.ts";
import { useMagicKeys, whenever } from "@vueuse/core";
import { Box, GalleryHorizontalEnd, Grid2X2, Search } from "lucide-vue-next";
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
const resourceRefStore = useResourceRefStore()
const { projectRefs, envRefs, appRefs } = storeToRefs(resourceRefStore);
const { userResources } = storeToRefs(userStore);

const { meta_j, ctrl_j } = useMagicKeys();
const anyJPressed = computed(() => meta_j.value || ctrl_j.value);
whenever(anyJPressed, () => {
  open.value = true;
});

// watch(projectRefs, (newRefs, oldRefs) => {
//   if (newRefs.length > 0 && newRefs[0].projectID !== oldRefs[0]?.projectID) {
//     if (userResources.value.projects.length === 0) {
//       userResources.value.projects.push(newRefs[0]);
//     } else {
//       const existingProjectIndex = userResources.value.projects.findIndex(
//         (project) => project.projectID === newRefs[0].projectID
//       );
//       if (existingProjectIndex === -1) {
//         userResources.value.projects.push(newRefs[0]);
//       } else {
//         userResources.value.projects[existingProjectIndex] = newRefs[0];
//       }
//     }
//   }
// });

// watch(envRefs, (newRefs, oldRefs) => {
//   if (newRefs.length > 0 && newRefs[0].envID !== oldRefs[0]?.envID) {
//     if (userResources.value.envs.length === 0) {
//       userResources.value.envs.push(newRefs[0]);
//     } else {
//       const existingEnvIndex = userResources.value.envs.findIndex(
//         (env) => env.envID === newRefs[0].envID
//       );
//       if (existingEnvIndex === -1) {
//         userResources.value.envs.push(newRefs[0]);
//       } else {
//         userResources.value.envs[existingEnvIndex] = newRefs[0];
//       }
//     }
//   }
// });

// watch(appRefs, (newRefs, oldRefs) => {
//   if (newRefs.length > 0 && newRefs[0].appID !== oldRefs[0]?.appID) {
//     if (userResources.value.apps.length === 0) {
//       userResources.value.apps.push(newRefs[0]);
//     } else {
//       const existingAppIndex = userResources.value.apps.findIndex(
//         (app) => app.appID === newRefs[0].appID
//       );
//       if (existingAppIndex === -1) {
//         userResources.value.apps.push(newRefs[0]);
//       } else {
//         userResources.value.apps[existingAppIndex] = newRefs[0];
//       }
//     }
//   }
// });


function handleSelect(ev: CustomEvent, resourceType: string, resourceID?: string) {
  // ev.preventDefault();
  open.value = false;
  // eslint-disable-next-line no-console
  console.log("Selected: ", ev.detail.value);

  if (resourceType === "project") {
    resourceRefStore.switchProject(resourceID);
    // 刷新页面
    // window.location.reload();
  } else if (resourceType === "env") {
    resourceRefStore.switchEnv(resourceID);
    router.push({ name: 'envPage', params: { id: resourceID } });
  } else if (resourceType === "app") {
    resourceRefStore.switchApp(resourceID);
    router.push({ name: 'appPage', params: { id: resourceID } });
  }
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger class="w-full">
      <SidebarMenuButton class="dark:text-white border flex items-center justify-between text-muted-foreground">
        <Search />
        <span class="justify-center flex-1">聚焦</span>
        <span
          class="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[12px] font-medium text-muted-foreground opacity-100"><kbd><span
              class="text-sm">⌘</span> J</kbd></span>
      </SidebarMenuButton>
    </DialogTrigger>

    <DialogPortal>
      <DialogOverlay class="bg-background/80 fixed inset-0 z-30" />
      <DialogContent
        class="fixed top-[15%] left-[50%] max-h-[85vh] w-[90vw] max-w-[24rem] translate-x-[-50%] text-sm rounded-xl overflow-hidden border border-muted-foreground/30 bg-card focus:outline-none z-[100]">
        <VisuallyHidden>
          <DialogTitle>Command Menu</DialogTitle>
          <DialogDescription>Search for command</DialogDescription>
        </VisuallyHidden>

        <ComboboxRoot :open="true">
          <ComboboxInput placeholder="输入搜索 ..."
            class="bg-transparent w-full px-4 py-3 outline-none placeholder-muted-foreground" @keydown.enter.prevent />

          <ComboboxContent class="border-t border-muted-foreground/30 p-2 max-h-[20rem] overflow-y-auto"
            @escape-key-down="open = false">
            <ComboboxEmpty class="text-center text-muted-foreground p-4">
              No results
            </ComboboxEmpty>
            <ScrollArea>
              <ComboboxGroup>
                <ComboboxLabel
                  class="px-4 inline-flex w-full items-center gap-4 text-muted-foreground font-semibold mt-3 mb-3">
                  <Box class="h-4 w-4" />
                  应用
                </ComboboxLabel>
                <!-- <RouterLink v-for="item in userResources.apps" :key="item.appID"
                  :to="{ name: 'appPage', params: { id: item.appID } }"> -->
                <ComboboxItem v-for="item in userResources.apps" :key="item.appID" :value="item"
                  class="cursor-default pl-12 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'app', item.appID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
                <!-- </RouterLink> -->
              </ComboboxGroup>
              <ComboboxGroup>
                <ComboboxLabel
                  class="px-4 inline-flex w-full items-center gap-4 text-muted-foreground font-semibold mt-3 mb-3">
                  <Grid2X2 class="h-4 w-4" />
                  环境
                </ComboboxLabel>
                <!-- <RouterLink v-for="item in userResources.envs" :key="item.envID"
                  :to="{ name: 'envPage', params: { id: item.envID } }"> -->
                <ComboboxItem v-for="item in userResources.envs" :key="item.envID" :value="item"
                  class="cursor-default pl-12 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect($event, 'env', item.envID)">
                  <span>{{ item.displayName }}</span>
                  <span class="text-xs text-muted-foreground font-mono">{{ item.slug }}</span>
                </ComboboxItem>
                <!-- </RouterLink> -->
              </ComboboxGroup>
              <ComboboxGroup>
                <ComboboxLabel
                  class="px-4 inline-flex w-full items-center gap-4 text-muted-foreground font-semibold mt-3 mb-3">
                  <GalleryHorizontalEnd class="h-4 w-4" />
                  项目
                </ComboboxLabel>
                <ComboboxItem v-for="item in userResources.projects" :key="item.projectID" :value="item"
                  class="cursor-default pl-12 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
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
