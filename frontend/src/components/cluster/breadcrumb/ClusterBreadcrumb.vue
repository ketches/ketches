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
import type { clusterRefModel } from '@/types/cluster'
import { Check, ChevronDown, ChevronRight, Grid2X2 } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const nodeID = router.currentRoute.value.params.nodeID as string

const props = defineProps({
    clusterRefs: {
        type: Array as () => clusterRefModel[],
        default: () => []
    }
})

const clusterHover = ref(false)

const selectedClusterRef = ref<clusterRefModel | null>(props.clusterRefs.length > 0 ? props.clusterRefs[0] : null)

async function onSwitchCluster(clusterID: string) {
    // TODO: Implement cluster switching logic
}
</script>

<template>
    <BreadcrumbItem v-if="clusterRefs.length > 0" @mouseenter="clusterHover = true" @mouseleave="clusterHover = false">
        <DropdownMenu>
            <DropdownMenuTrigger class="flex items-center gap-1">
                <Button variant="ghost" size="sm">
                    <Grid2X2 />
                    <span>{{ selectedClusterRef?.displayName || '选择集群' }}</span>
                    <ChevronDown v-if="clusterHover" />
                    <ChevronRight v-else-if="selectedClusterRef" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem v-if="selectedClusterRef" :key="selectedClusterRef.clusterID" :disabled="!nodeID">
                    <RouterLink :to="{ name: 'app' }" v-slot="{ navigate, href }"
                        class="flex items-center gap-2 w-full">
                        <Check class="text-green-500 font-medium" />
                        <span :href="href" @click="navigate">{{ selectedClusterRef.displayName }}</span>
                        <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                            selectedClusterRef.slug }}
                        </Badge>
                    </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuItem
                    v-for="clusterRef in clusterRefs.filter(clusterRef => clusterRef.clusterID !== selectedClusterRef?.clusterID)"
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
