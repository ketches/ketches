<script setup lang="ts">
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import {
    useSidebar
} from '@/components/ui/sidebar';
import SidebarMenu from "@/components/ui/sidebar/SidebarMenu.vue";
import SidebarMenuButton from "@/components/ui/sidebar/SidebarMenuButton.vue";
import SidebarMenuItem from "@/components/ui/sidebar/SidebarMenuItem.vue";
import { useResourceRefStore } from '@/stores/resourceRefStore';
import { useUserStore } from "@/stores/userStore";
import { Check, ChevronsUpDown, Cog, GalleryHorizontalEnd, Plus } from 'lucide-vue-next';
import { storeToRefs } from 'pinia';
import { ref } from 'vue';
import ProjectForm from '../project/ProjectForm.vue';
import Button from '../ui/button/Button.vue';

const { isMobile } = useSidebar();
const userStore = storeToRefs(useUserStore());
const userInfo = userStore.user;

const noProjects = ref(false);

const resourceRefStore = useResourceRefStore()
const { activeProjectRef, projectRefs } = storeToRefs(resourceRefStore)

function onSwitchProject(projectID: string) {
    resourceRefStore.switchProject(projectID!);
}

const openProjectForm = ref(false);
</script>

<template>
    <SidebarMenu>
        <SidebarMenuItem v-if="userInfo?.role === 'admin'">
            <SidebarMenuButton size="lg" as-child>
                <a href="#">
                    <img src="/ketches.svg" alt="Ketches Logo" style="height: 100%" />
                    <div class="flex flex-col gap-0.5 leading-none">
                        <span class="font-semibold">Ketches 平台</span>
                        <span class="text-muted-foreground">v1.0.0</span>
                    </div>
                </a>
            </SidebarMenuButton>
        </SidebarMenuItem>
        <SidebarMenuItem v-else-if="userInfo?.role === 'user'">
            <DropdownMenu v-if="projectRefs.length > 0">
                <DropdownMenuTrigger as-child>
                    <SidebarMenuButton size="lg"
                        class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
                        <div
                            class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                            <GalleryHorizontalEnd class="size-4" />
                        </div>
                        <div class="grid flex-1 text-left text-sm leading-tight">
                            <span class="truncate font-medium">
                                {{ activeProjectRef?.displayName || activeProjectRef?.slug }}
                            </span>
                            <span class="truncate text-xs">{{ activeProjectRef?.slug }}</span>
                        </div>
                        <ChevronsUpDown class="ml-auto" />
                    </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent class="w-[--reka-dropdown-menu-trigger-width] min-w-56 rounded-lg" align="start"
                    :side="isMobile ? 'bottom' : 'right'" :side-offset="4">
                    <DropdownMenuLabel class="text-xs text-muted-foreground">
                        项目
                    </DropdownMenuLabel>
                    <DropdownMenuItem v-for="projectRef in projectRefs" :key="projectRef.slug" class="gap-2 p-2"
                        @click="onSwitchProject(projectRef.projectID)"
                        :disabled="activeProjectRef?.projectID === projectRef.projectID">
                        <div class="flex size-6 items-center justify-center rounded-sm border">
                            <GalleryHorizontalEnd class="size-3.5 shrink-0" />
                        </div>
                        {{ projectRef.displayName }}
                        <Check v-if="activeProjectRef?.projectID === projectRef.projectID" class="ml-auto right-0" />
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem class="gap-2 p-2" @click="openProjectForm = true">
                        <div class="flex size-6 items-center justify-center rounded-sm border">
                            <Plus class="size-3.5" />
                        </div>
                        <span>创建项目</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem class="gap-2 p-2">
                        <div class="flex size-6 items-center justify-center rounded-sm border">
                            <Cog class="size-3.5" />
                        </div>
                        <span>项目设置</span>
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <SidebarMenuButton v-if="noProjects" size="lg" class="text-muted-foreground">
                <Button variant="default" class="w-full" @click="openProjectForm = true">
                    <Plus />
                    创建项目
                </Button>
            </SidebarMenuButton>
        </SidebarMenuItem>
    </SidebarMenu>
    <ProjectForm v-model="openProjectForm" />
</template>
