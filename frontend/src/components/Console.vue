<script setup lang="ts">
import Sidebar from './Sidebar.vue';
import SidebarProvider from './ui/sidebar/SidebarProvider.vue';

</script>

<template>
  <SidebarProvider>
    <Sidebar />
    <RouterView />
  </SidebarProvider>
</template>