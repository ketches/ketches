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
import { Check, ChevronDown, Package } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

const appHover = ref(false)

const router = useRouter()
const userStore = useUserStore()
const { activeAppRef } = storeToRefs(userStore)

async function onSwitchApp(appID: string) {
    userStore.activateApp(appID)
    router.push({ name: 'app-page', params: { id: appID } })
}

</script>

<template>
    <BreadcrumbItem @mouseenter="appHover = true" @mouseleave="appHover = false">
        <DropdownMenu>
            <DropdownMenuTrigger class="flex items-center gap-1">
                <Button variant="ghost" size="sm">
                    <Package />
                    <span>{{ activeAppRef?.displayName }}</span>
                    <ChevronDown v-if="appHover" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem v-if="activeAppRef" :key="activeAppRef.appID" disabled>
                    <RouterLink :to="{ name: 'app' }" v-slot="{ navigate, href }"
                        class="flex items-center gap-2 w-full">
                        <Check class="text-green-500 font-medium" />
                        <span :href="href" @click="navigate">{{ activeAppRef.displayName }}</span>
                        <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                            activeAppRef.slug }}
                        </Badge>
                    </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuItem
                    v-for="appRef in userStore.getCurrentAppRefs.filter(app => app.appID !== activeAppRef?.appID)"
                    @click="onSwitchApp(appRef.appID)" :key="appRef.appID">
                    <div class="h-4 w-4" />
                    <span>{{ appRef.displayName }}</span>
                    <Badge variant="secondary" class="text-xs text-muted-foreground font-mono ml-auto right-0">{{
                        appRef.slug }}
                    </Badge>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    </BreadcrumbItem>
</template>
