<script setup lang="ts">
import { Toaster } from '@/components/ui/sonner';
// import { storeToRefs } from 'pinia';
// import { onMounted, watch } from 'vue';
// import { useRoute, useRouter } from 'vue-router';
// import 'vue-sonner/style.css';
// import { useUserStore } from './stores/userStore';

// const route = useRoute();
// const router = useRouter();

// const userStore = useUserStore();
// const { user, userResources, activeProjectRef } = storeToRefs(userStore);

// watch(
//   () => route.name,
//   async (routeName) => {
//     if (!routeName) {
//       return;
//     }

//     if (!user.value) {
//       await userStore.initUser();
//     }

//     await userStore.fetchUserResourceRefs();
//     await router.isReady();

//     if (user.value) {
//       switch (routeName) {
//         case "appPage":
//           const appID = route.params.id as string;
//           userStore.activateApp(appID);
//           break;
//         case "envPage":
//           const envID = route.params.id as string;
//           userStore.activateEnv(envID);
//           break;
//         case "home":
//           if (user.value.role === "admin") {
//             router.push({ name: 'admin' });
//           } else {
//             console.log("Ensuring active project for user");

//             userStore.ensureActiveProject();
//             router.push({ name: 'user' });
//           }
//           break;
//         default:
//           userStore.ensureActiveProject();
//           break;
//       }
//     }
//   },
//   { immediate: true }
// );

// watch(activeProjectRef, (newActiveProjectRef, oldActiveProjectRef) => {
//   if (oldActiveProjectRef && newActiveProjectRef?.projectID !== oldActiveProjectRef.projectID) {
//     const currentPath = router.currentRoute.value.path;
//     const userRoutes = router.getRoutes().find(r => r.name === 'user')?.children || [];

//     const targetRoute = userRoutes.find(route => currentPath.startsWith(`/${route.path}`));

//     if (targetRoute && targetRoute.name) {
//       router.push({ name: targetRoute.name });
//     } else {
//       router.push({ name: 'home' });
//     }
//   }
// });

// onMounted(() => {
//   const viewport = document.querySelector('.xterm-viewport');
//   let timer: number | undefined;
//   if (viewport) {
//     viewport.addEventListener('scroll', () => {
//       viewport.classList.add('scrolling');
//       clearTimeout(timer);
//       timer = setTimeout(() => {
//         viewport.classList.remove('scrolling');
//       }, 1200); // 1.2秒后隐藏
//     });
//   }
// });
</script>

<template>
  <Toaster richColors />
  <RouterView />
</template>