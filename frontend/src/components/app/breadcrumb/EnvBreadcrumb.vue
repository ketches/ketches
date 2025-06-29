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
import { Check, ChevronDown, ChevronRight, Grid2X2 } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const envHover = ref(false)

const userStore = useUserStore()
const { activeEnvRef, activeAppRef } = storeToRefs(userStore)

async function onSwitchEnv(envID: string) {
    await userStore.activateEnv(envID!)
    router.push({ name: 'app' });
}
</script>

<template>
    <BreadcrumbItem v-if="userStore.getCurrentEnvRefs.length > 0" @mouseenter="envHover = true"
        @mouseleave="envHover = false">
        <DropdownMenu>
            <DropdownMenuTrigger class="flex items-center gap-1">
                <Button variant="ghost" size="sm">
                    <Grid2X2 />
                    <span>{{ activeEnvRef?.displayName || '选择环境' }}</span>
                    <ChevronDown v-if="envHover" />
                    <ChevronRight v-else-if="activeAppRef" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem v-if="activeEnvRef" :key="activeEnvRef.envID" :disabled="!activeAppRef">
                    <RouterLink :to="{ name: 'app' }" v-slot="{ navigate, href }"
                        class="flex items-center gap-2 w-full">
                        <Check class="text-green-500 font-medium" />
                        <span :href="href" @click="navigate">{{ activeEnvRef.displayName }}</span>
                        <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                            activeEnvRef.slug }}
                        </Badge>
                    </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuItem
                    v-for="envRef in userStore.getCurrentEnvRefs.filter(env => env.envID !== activeEnvRef?.envID)"
                    @click="onSwitchEnv(envRef.envID)" :key="envRef.envID">
                    <div class="h-4 w-4" />
                    <span>{{ envRef.displayName }}</span>
                    <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                        envRef.slug }}
                    </Badge>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    </BreadcrumbItem>
</template>
