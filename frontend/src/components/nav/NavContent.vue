<script setup lang="ts">
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { useUserStore } from '@/stores/userStore';
import { Boxes, EllipsisVertical, Plus, Telescope } from "lucide-vue-next";
import { storeToRefs } from 'pinia';
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';
import CreateCluster from '../cluster/CreateCluster.vue';
import Collapsible from "../ui/collapsible/Collapsible.vue";
import {
    SidebarGroup,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "../ui/sidebar";
import SidebarMenuAction from '../ui/sidebar/SidebarMenuAction.vue';
import Spot from './Spot.vue';
import { userNavContents as rawNavContents } from './navContent';

const { isMobile } = useSidebar();

const { user } = storeToRefs(useUserStore());

const openClusterForm = ref(false);

const route = useRoute();
const userNavContents = computed(() =>
    rawNavContents.map(item => ({
        ...item,
        isActive: route.matched.some(r => r.name === item.route.name)
    }))
);
</script>

<template>
    <SidebarGroup v-if="user">
        <SidebarMenu v-if="user?.role === 'admin'">
            <SidebarMenuItem class="pb-4">
                <Spot />
            </SidebarMenuItem>
            <SidebarMenuItem>
                <RouterLink :to="{ name: 'admin-overview' }" class="flex w-full">
                    <SidebarMenuButton>
                        <Telescope />
                        <span>总览</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
            <Collapsible as-child class="group/collapsible">
                <SidebarMenuItem>
                    <RouterLink :to="{ name: 'cluster' }" class="w-full">
                        <SidebarMenuButton>
                            <Boxes class="h-4 w-4" />
                            集群
                        </SidebarMenuButton>
                        <DropdownMenu>
                            <DropdownMenuTrigger as-child>
                                <SidebarMenuAction show-on-hover>
                                    <EllipsisVertical />
                                </SidebarMenuAction>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent class="w-48 rounded-lg" :side="isMobile ? 'bottom' : 'right'"
                                :align="isMobile ? 'end' : 'start'">
                                <DropdownMenuItem @click="openClusterForm = true" @select.prevent>
                                    <Plus class="text-muted-foreground" />
                                    <span>创建集群</span>
                                </DropdownMenuItem>
                                <CreateCluster v-model:open="openClusterForm" />
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </RouterLink>
                </SidebarMenuItem>
            </Collapsible>
        </SidebarMenu>
        <SidebarMenu v-else-if="user?.role === 'user'">
            <SidebarMenuItem class="pb-4">
                <Spot />
            </SidebarMenuItem>
            <SidebarMenuItem v-for="item in userNavContents" :key="item.title">
                <RouterLink :to="item.route" class="flex items-center gap-2">
                    <SidebarMenuButton :tooltip="item.title" :is-active="item.isActive">
                        <component :is="item.icon" class="h-4 w-4" />
                        <span class="flex-1">{{ item.title }}</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
        </SidebarMenu>
    </SidebarGroup>
</template>
