<script setup lang="ts">
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { useUserStore } from '@/stores/userStore';
import { Box, Boxes, EllipsisVertical, Grid2X2, Plus, Telescope, UsersRound } from "lucide-vue-next";
import { storeToRefs } from 'pinia';
import { ref } from 'vue';
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

const { isMobile } = useSidebar();

const { user } = storeToRefs(useUserStore());

const openClusterForm = ref(false);
</script>

<template>
    <SidebarGroup v-if="user">
        <SidebarMenu v-if="user?.role === 'admin'">
            <SidebarMenuItem class="pb-4">
                <Spot />
            </SidebarMenuItem>
            <SidebarMenuItem>
                <SidebarMenuButton>
                    <Telescope />
                    <RouterLink :to="{ name: 'admin-overview' }" class="flex w-full">
                        <span>总览</span>
                    </RouterLink>
                </SidebarMenuButton>
            </SidebarMenuItem>
            <Collapsible as-child class="group/collapsible">
                <SidebarMenuItem>
                    <SidebarMenuButton>
                        <Boxes class="h-4 w-4" />
                        <RouterLink :to="{ name: 'cluster' }" class="w-full">
                            集群管理
                        </RouterLink>
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
                </SidebarMenuItem>
            </Collapsible>
        </SidebarMenu>
        <SidebarMenu v-else-if="user?.role === 'user'">
            <SidebarMenuItem class="pb-4">
                <Spot />
            </SidebarMenuItem>
            <SidebarMenuItem>
                <RouterLink :to="{ name: 'overview' }" class="flex items-center gap-2">
                    <SidebarMenuButton tooltip="总览">
                        <Telescope class="h-4 w-4" />
                        <span class="flex-1">总览</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
                <RouterLink :to="{ name: 'env' }" class="flex items-center gap-2">
                    <SidebarMenuButton tooltip="环境">
                        <Grid2X2 class="h-4 w-4" />
                        <span class="flex-1">环境</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
                <RouterLink :to="{ name: 'app' }" class="flex items-center gap-2">
                    <SidebarMenuButton tooltip="应用">
                        <Box class="h-4 w-4" />
                        <span class="flex-1">应用</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
                <RouterLink :to="{ name: 'member' }" class="flex items-center gap-2">
                    <SidebarMenuButton tooltip="成员">
                        <UsersRound class="h-4 w-4" />
                        <span class="flex-1">成员</span>
                    </SidebarMenuButton>
                </RouterLink>
            </SidebarMenuItem>
        </SidebarMenu>
    </SidebarGroup>
</template>
