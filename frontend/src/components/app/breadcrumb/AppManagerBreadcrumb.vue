<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from '@/components/ui/breadcrumb';
import { useResourceRefStore } from '@/stores/resourceRefStore';
import { storeToRefs } from 'pinia';
import AppBreadcrumb from './AppBreadcrumb.vue';
import EnvBreadcrumb from './EnvBreadcrumb.vue';
defineProps({
  appID: {
    type: String,
    default: '',
  },
});

const resourceRefStore = useResourceRefStore();
const { envRefs } = storeToRefs(resourceRefStore);
</script>

<template>
  <Breadcrumb>
    <BreadcrumbList>
      <EnvBreadcrumb v-if="envRefs.length > 0" />
      <!-- <BreadcrumbList v-else> -->
      <BreadcrumbItem v-else>应用管理</BreadcrumbItem>
      <!-- </BreadcrumbList> -->
      <AppBreadcrumb v-if="appID" :appID="appID" />
    </BreadcrumbList>
  </Breadcrumb>
</template>