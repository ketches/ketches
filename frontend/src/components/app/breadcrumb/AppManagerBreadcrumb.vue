<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from '@/components/ui/breadcrumb';
import { useUserStore } from '@/stores/userStore';
import AppBreadcrumb from './AppBreadcrumb.vue';
import EnvBreadcrumb from './EnvBreadcrumb.vue';
defineProps({
  appID: {
    type: String,
    default: '',
  },
});

const userStore = useUserStore();
</script>

<template>
  <Breadcrumb>
    <BreadcrumbList>
      <BreadcrumbItem class="font-semibold">应用</BreadcrumbItem>
      <EnvBreadcrumb v-if="userStore.getCurrentEnvRefs.length > 0" />
      <AppBreadcrumb v-if="appID" :appID="appID" />
    </BreadcrumbList>
  </Breadcrumb>
</template>
