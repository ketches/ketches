<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  useSidebar
} from "@/components/ui/sidebar";
import { PanelLeftClose, PanelLeftOpen } from "lucide-vue-next";
import { computed } from "vue";
import { useRoute } from "vue-router";
import ClusterBreadcrumb from "./breadcrumb/ClusterManagerBreadcrumb.vue";

const { toggleSidebar, open } = useSidebar();

const route = useRoute();

const id = computed(() => {
  return route.params.id as string;
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
        <ClusterBreadcrumb />
      </div>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
      集群: {{ id }}
      <RouterView />
    </div>
  </SidebarInset>
</template>
