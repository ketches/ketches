<script setup lang="ts">
import SidebarProvider from '@/components/ui/sidebar/SidebarProvider.vue';
import { useUserStore } from '@/stores/userStore';
import Cookies from 'js-cookie';
import { storeToRefs } from 'pinia';
import { watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Toaster } from 'vue-sonner';
import 'vue-sonner/style.css';
import Sidebar from './Sidebar.vue';

const defaultOpen = Cookies.get('sidebar_state') === 'true' || Cookies.get('sidebar_state') === undefined;

const route = useRoute();
const router = useRouter();

const userStore = useUserStore();
const { user, userResources, activeProjectRef } = storeToRefs(userStore);

watch(
  () => route.name,
  async (routeName) => {
    if (!routeName) {
      return;
    }

    if (!user.value) {
      await userStore.initUser();
    }

    await userStore.fetchUserResourceRefs();
    await router.isReady();

    if (user.value) {
      switch (routeName) {
        case "appPage":
          const appID = route.params.id as string;
          userStore.activateApp(appID);
          break;
        case "envPage":
          const envID = route.params.id as string;
          userStore.activateEnv(envID);
          break;
        case "home":
          if (user.value.role === "admin") {
            router.push({ name: 'admin' });
          } else {
            console.log("Ensuring active project for user");

            userStore.ensureActiveProject();
            router.push({ name: 'user' });
          }
          break;
        default:
          userStore.ensureActiveProject();
          break;
      }
    }
  },
  { immediate: true }
);

watch(activeProjectRef, (newActiveProjectRef, oldActiveProjectRef) => {
  if (oldActiveProjectRef && newActiveProjectRef?.projectID !== oldActiveProjectRef.projectID) {
    const currentPath = router.currentRoute.value.path;
    const userRoutes = router.getRoutes().find(r => r.name === 'user')?.children || [];

    const targetRoute = userRoutes.find(route => currentPath.startsWith(`/${route.path}`));

    if (targetRoute && targetRoute.name) {
      router.push({ name: targetRoute.name });
    } else {
      router.push({ name: 'home' });
    }
  }
});
</script>

<template>
  <Toaster richColors />
  <SidebarProvider :default-open="defaultOpen">
    <Sidebar />
    <RouterView />
  </SidebarProvider>
</template>