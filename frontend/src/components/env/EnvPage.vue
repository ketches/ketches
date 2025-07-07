<script setup lang="ts">
import { getEnv } from "@/api/env";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  useSidebar
} from "@/components/ui/sidebar";
import { useUserStore } from "@/stores/userStore";
import type { envModel } from "@/types/env";
import { PanelLeftClose, PanelLeftOpen } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { ref, watch } from "vue";
import { useRoute } from "vue-router";
import EnvManagerBreadcrumb from "./breadcrumb/EnvManagerBreadcrumb.vue";


const { toggleSidebar, open } = useSidebar();
const route = useRoute();
const envID = route.params.id as string;

const userStore = useUserStore()
const { activeEnvRef } = storeToRefs(userStore);

const env = ref<envModel | null>(null);

async function fetchEnvInfo(envID?: string) {
  if (envID) {
    env.value = await getEnv(envID)
  }
}

watch(activeEnvRef, async (newEnvRef) => {
  if (newEnvRef && newEnvRef.envID !== env.value?.envID) {
    await fetchEnvInfo(newEnvRef.envID);
  }
});

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
        <EnvManagerBreadcrumb :envID="envID" />
      </div>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
      环境：{{ env?.envID }}
    </div>
  </SidebarInset>
</template>
