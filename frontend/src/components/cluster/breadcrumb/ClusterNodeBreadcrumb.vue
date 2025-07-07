<script lang="ts" setup>
import Badge from '@/components/ui/badge/Badge.vue'
import {
    BreadcrumbItem
} from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useUserStore } from '@/stores/userStore'
import { Check, ChevronDown, Server } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

const clusterNodeHover = ref(false)

const router = useRouter()
const userStore = useUserStore()
const { activeClusterNodeRef } = storeToRefs(userStore)

async function onSwitchClusterNode(clusterID: string, nodeName: string) {
    userStore.activateClusterNode(clusterID, nodeName)
    await router.push({ name: 'cluster-node-page', params: { id: clusterID } })
}

</script>

<template>
    <BreadcrumbItem @mouseenter="clusterNodeHover = true" @mouseleave="clusterNodeHover = false">
        <DropdownMenu>
            <DropdownMenuTrigger class="flex items-center gap-1">
                <Button variant="ghost" size="sm">
                    <Server />
                    <span>{{ activeClusterNodeRef?.nodeName }}</span>
                    <ChevronDown v-if="clusterNodeHover" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem v-if="activeClusterNodeRef" :key="activeClusterNodeRef.nodeName" disabled>
                    <RouterLink :to="{ name: 'app' }" v-slot="{ navigate, href }"
                        class="flex items-center gap-2 w-full">
                        <Check class="text-green-500 font-medium" />
                        <span :href="href" @click="navigate">{{ activeClusterNodeRef.nodeName }}</span>
                        <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                            activeClusterNodeRef.nodeIP }}
                        </Badge>
                    </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuItem
                    v-for="clusterNode in userStore.getCurrentClusterNodeRefs.filter(clusterNode => clusterNode.nodeName !== activeClusterNodeRef?.nodeName)"
                    @click="onSwitchClusterNode(clusterNode.clusterID, clusterNode.nodeName)"
                    :key="clusterNode.nodeName">
                    <div class="h-4 w-4" />
                    <span>{{ clusterNode.nodeName }}</span>
                    <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                        clusterNode.nodeIP }}
                    </Badge>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    </BreadcrumbItem>
</template>
