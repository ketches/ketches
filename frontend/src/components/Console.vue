<script setup lang="ts">
import Cookies from 'js-cookie';
import Sidebar from './Sidebar.vue';
import SidebarProvider from './ui/sidebar/SidebarProvider.vue';

const defaultOpen = Cookies.get('sidebar_state') === 'true'
</script>

<template>
  <SidebarProvider :default-open="defaultOpen">
    <Sidebar />
    <RouterView />
  </SidebarProvider>
</template>