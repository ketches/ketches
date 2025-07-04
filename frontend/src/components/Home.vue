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
const { user, userResources, activeProjectRef, activeClusterRef, activeClusterNodeRef } = storeToRefs(userStore);

// const clusterStore = useClusterStore();
// const { activeClusterRef, activeClusterNodeRef } = storeToRefs(clusterStore);

watch(
  () => route.name,
  async (routeName) => {
    if (!routeName) {
      return;
    }

    if (!user.value) {
      await userStore.initUser();
    }

    if (user.value.role === 'admin') {
      await userStore.fetchAdminResourceRefs();
    } else {
      await userStore.fetchUserResourceRefs();
    }

    await router.isReady();

    if (user.value) {
      if (user.value.role === 'admin') {
        switch (routeName) {
          case "home":
            router.push({ name: 'admin' });
            break;
          case "appPage":
            const appID = route.params.id as string;
            userStore.activateApp(appID);
            break;
          case "envPage":
            const envID = route.params.id as string;
            userStore.activateEnv(envID);
            break;
          case "clusterPage":
            {
              const clusterID = route.params.id as string;
              userStore.activateCluster(clusterID);
              if (!activeClusterRef.value) {
                router.push({ name: 'cluster' });
              }
            }
            break;
          case "clusterNodePage":
            {
              const clusterID = route.params.id as string;
              const nodeName = route.params.nodeName as string;
              userStore.activateClusterNode(clusterID, nodeName);
              if (!activeClusterRef.value) {
                router.push({ name: 'cluster' });
              } else if (!userStore.activeClusterNodeRef) {
                router.push({ name: 'clusterPage', params: { id: clusterID } });
              }
            }
        }
      } else {
        switch (routeName) {
          case "home":
            router.push({ name: 'user' });
            break;
          case "appPage":
            const appID = route.params.id as string;
            userStore.activateApp(appID);
            break;
          case "envPage":
            const envID = route.params.id as string;
            userStore.activateEnv(envID);
            break;
          default:
            if (user.value.role === "user") {
              userStore.ensureActiveProject();
            }
            break;
        }
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