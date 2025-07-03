<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from '@/components/ui/breadcrumb';
import type { clusterRefModel } from '@/types/cluster';
import ClusterBreadcrumb from './ClusterBreadcrumb.vue';
import NodeBreadcrumb from './NodeBreadcrumb.vue';

const props = defineProps({
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

console.log(props.clusterRefs);

</script>

<template>
  <Breadcrumb>
    <BreadcrumbList>
      <BreadcrumbItem class="font-semibold">集群</BreadcrumbItem>
      <ClusterBreadcrumb v-if="clusterID" :clusterID="clusterID" :clusterRefs="props.clusterRefs" />
      <!-- <BreadcrumbList v-else> -->
      <!-- </BreadcrumbList> -->
      <NodeBreadcrumb v-if="nodeName" :nodeName="nodeName" />
    </BreadcrumbList>
  </Breadcrumb>
</template>