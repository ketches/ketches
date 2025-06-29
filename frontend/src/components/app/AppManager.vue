<script setup lang="ts">
import {
  useSidebar
} from "@/components/ui/sidebar"
import { useResourceRefStore } from '@/stores/resourceRefStore'
import { PanelLeftClose, PanelLeftOpen, Plus } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { ref, watch } from 'vue'
import CreateEnv from "../env/CreateEnv.vue"
import Button from '../ui/button/Button.vue'
import Separator from '../ui/separator/Separator.vue'
import SidebarInset from '../ui/sidebar/SidebarInset.vue'
import AppList from './AppList.vue'
import AppManagerBreadcrumb from './breadcrumb/AppManagerBreadcrumb.vue'
const { toggleSidebar, open } = useSidebar();

const resourceRefStore = useResourceRefStore()
const { envRefs } = storeToRefs(resourceRefStore)

const noEnvs = ref(false)
const hasEnvs = ref(false)

watch(envRefs, (newEnvRefs) => {
  noEnvs.value = newEnvRefs.length === 0
  hasEnvs.value = newEnvRefs.length > 0
}, { immediate: true })

const openEnvForm = ref(false)
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
        <AppManagerBreadcrumb />
      </div>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div v-if="noEnvs"
        class="flex flex-col flex-grow text-balance text-center text-sm text-muted-foreground justify-center items-center">
        <span class="block mb-2">当前项目还没有环境，让我们先来创建一个吧！</span>
        <Button variant="default" class="my-4" @click="openEnvForm = true">
          <Plus />
          创建环境
        </Button>
      </div>
      <AppList v-if="hasEnvs" />
    </div>
  </SidebarInset>
  <CreateEnv v-model="openEnvForm" />
</template>
