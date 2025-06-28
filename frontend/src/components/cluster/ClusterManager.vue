<script setup lang="ts">
import { fetchClusterRefs } from "@/api/cluster"
import {
  useSidebar
} from "@/components/ui/sidebar"
import type { clusterRefModel } from "@/types/cluster"
import { PanelLeftClose, PanelLeftOpen } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import Button from '../ui/button/Button.vue'
import Separator from '../ui/separator/Separator.vue'
import SidebarInset from '../ui/sidebar/SidebarInset.vue'
import ClusterManagerBreadcrumb from './breadcrumb/ClusterManagerBreadcrumb.vue'
import ClusterList from './ClusterList.vue'
const { toggleSidebar, open } = useSidebar();

const noClusters = ref(false)

const clusterRefs = ref<clusterRefModel[]>([])

onMounted(async () => {
  clusterRefs.value = await fetchClusterRefs()
  noClusters.value = clusterRefs.value.length === 0
})

const openClusterForm = ref(false)
</script>

<template>
  <SidebarInset>
    <header
      class="flex h-12 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
      <div class="flex items-center gap-2 px-4">
        <Button variant="ghost" @click="toggleSidebar" class="h-8 w-8 text-muted-foreground hover:text-primary">
          <PanelLeftOpen v-if="!open" />
          <PanelLeftClose v-else />
        </Button>
        <Separator orientation="vertical" class="mr-2 h-4" />
        <ClusterManagerBreadcrumb :clusterRefs="clusterRefs" />
      </div>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
      <!-- <div v-if="noClusters"
        class="flex flex-col flex-grow text-balance text-center text-sm text-muted-foreground justify-center items-center">
        <span class="block mb-2">当前项目还没有集群，让我们先来创建一个吧！</span>
        <Button variant="default" class="my-4" @click="openClusterForm = true">
          <Plus />
          创建集群
        </Button>
      </div> -->
      <ClusterList />
    </div>
  </SidebarInset>
  <!-- <CreateCluster v-model="openClusterForm" @cluster-created=""/> -->
</template>
