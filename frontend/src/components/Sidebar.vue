<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import NavContent from "./nav/NavContent.vue";
import NavFooter from "./nav/NavFooter.vue";
import NavHeader from "./nav/NavHeader.vue";

const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});


</script>

<template>
  <Sidebar v-bind="props">
    <SidebarHeader>
      <NavHeader />
    </SidebarHeader>
    <SidebarContent>
      <NavContent />
    </SidebarContent>
    <SidebarFooter>
      <NavFooter />
    </SidebarFooter>
    <SidebarRail />
  </Sidebar>
</template>
