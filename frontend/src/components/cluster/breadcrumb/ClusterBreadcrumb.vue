<script lang="ts" setup>
import { Badge } from '@/components/ui/badge'
import {
    BreadcrumbItem
} from '@/components/ui/breadcrumb'
import Button from '@/components/ui/button/Button.vue'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useUserStore } from '@/stores/userStore'
import { Boxes, Check, ChevronDown, ChevronRight } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const clusterHover = ref(false)

const userStore = useUserStore()
const { adminResources, activeClusterRef, activeClusterNodeRef } = storeToRefs(userStore)

function onSwitchCluster(clusterID: string) {
    userStore.activateCluster(clusterID)
    router.push({ name: 'clusterPage', params: { id: clusterID } })
}
</script>

<template>
    <BreadcrumbItem v-if="adminResources?.clusters.length > 0" @mouseenter="clusterHover = true"
        @mouseleave="clusterHover = false">
        <DropdownMenu>
            <DropdownMenuTrigger class="flex items-center gap-1">
                <Button variant="ghost" size="sm">
                    <Boxes />
                    <span>{{ activeClusterRef?.displayName || '选择集群' }}</span>
                    <ChevronDown v-if="clusterHover" />
                    <ChevronRight v-else-if="activeClusterRef" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem v-if="activeClusterRef" :key="activeClusterRef.clusterID"
                    :disabled="!activeClusterNodeRef">
                    <RouterLink :to="{ name: 'app' }" v-slot="{ navigate, href }"
                        class="flex items-center gap-2 w-full">
                        <Check class="text-green-500 font-medium" />
                        <span :href="href" @click="navigate">{{ activeClusterRef.displayName }}</span>
                        <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                            activeClusterRef.slug }}
                        </Badge>
                    </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuItem
                    v-for="clusterRef in adminResources?.clusters.filter(clusterRef => clusterRef.clusterID !== activeClusterRef?.clusterID)"
                    @click="onSwitchCluster(clusterRef.clusterID)" :key="clusterRef.clusterID">
                    <div class="h-4 w-4" />
                    <span>{{ clusterRef.displayName }}</span>
                    <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                        clusterRef.slug }}
                    </Badge>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    </BreadcrumbItem>
</template>
