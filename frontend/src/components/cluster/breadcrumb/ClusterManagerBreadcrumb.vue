<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from '@/components/ui/breadcrumb';
import ClusterBreadcrumb from './ClusterBreadcrumb.vue';
import ClusterNodeBreadcrumb from './ClusterNodeBreadcrumb.vue';

const props = defineProps({
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
      <BreadcrumbItem class="font-semibold">集群</BreadcrumbItem>
      <ClusterBreadcrumb v-if="clusterID" />
      <ClusterNodeBreadcrumb v-if="nodeName" />
    </BreadcrumbList>
  </Breadcrumb>
</template>