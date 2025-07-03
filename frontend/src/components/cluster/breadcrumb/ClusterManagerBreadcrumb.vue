<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from '@/components/ui/breadcrumb';
import type { clusterRefModel } from '@/types/cluster';
import ClusterBreadcrumb from './ClusterBreadcrumb.vue';
import NodeBreadcrumb from './NodeBreadcrumb.vue';
defineProps({
  clusterRefs: {
    type: Array as () => clusterRefModel[],
    default: () => [],
  },
  clusterID: {
    type: String,
    default: '',
  },
  nodeName: {
    type: String,
    default: '',
  },
});

</script>

<template>
  <Breadcrumb>
    <BreadcrumbList>
      <ClusterBreadcrumb v-if="clusterID" :clusterID="clusterID" :clusterRefs="clusterRefs" />
      <!-- <BreadcrumbList v-else> -->
      <BreadcrumbItem v-else class="font-semibold">集群</BreadcrumbItem>
      <!-- </BreadcrumbList> -->
      <NodeBreadcrumb v-if="nodeName" :nodeName="nodeName" />
    </BreadcrumbList>
  </Breadcrumb>
</template>