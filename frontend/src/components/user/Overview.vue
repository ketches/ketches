<script setup lang="ts">
import { fetchProjectStatistics } from '@/api/project';
import {
    Card,
    CardContent,
    CardHeader,
    CardTitle
} from '@/components/ui/card';
import { SidebarInset, useSidebar } from '@/components/ui/sidebar';
import { useUserStore } from '@/stores/userStore';
import { Grid2X2, Network, Package, PanelLeftClose, PanelLeftOpen, Users } from 'lucide-vue-next';
import { storeToRefs } from 'pinia';
import { onMounted, ref, watch, type Component } from 'vue';
import { Breadcrumb, BreadcrumbItem, BreadcrumbList } from '../ui/breadcrumb';
import { Button } from '../ui/button';
import { Separator } from '../ui/separator';

const { toggleSidebar, open } = useSidebar();

interface statisticsItme {
    label: string;
    icon: Component;
    total: number;
    description: string;
}

const userStore = useUserStore()
const { activeProjectRef } = storeToRefs(userStore)

const fetchedStatistics = ref<statisticsItme[]>([]);

async function fetchStatistics(projectID: string) {
    const resp = await fetchProjectStatistics(projectID);
    if (resp) {
        fetchedStatistics.value = [
            {
                label: '环境',
                icon: Grid2X2,
                total: resp.totalEnvs,
                description: '环境总数'
            },
            {
                label: '应用',
                icon: Package,
                total: resp.totalApps,
                description: '应用总数'
            },
            {
                label: '网关',
                icon: Network,
                total: resp.totalAppGateways,
                description: '网关总数'
            },
            {
                label: '成员',
                icon: Users,
                total: resp.totalMembers,
                description: '成员总数'
            },
        ];
    } else {
        fetchedStatistics.value = [];
    }
}

onMounted(async () => {
    if (activeProjectRef.value) {
        await fetchStatistics(activeProjectRef.value.projectID);
    } else {
        fetchedStatistics.value = [];
    }
});

watch(activeProjectRef, async (newProjectRef) => {
    if (newProjectRef) {
        await fetchStatistics(newProjectRef.projectID);
    } else {
        fetchedStatistics.value = [];
    }
}, { immediate: true });

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
                <Breadcrumb>
                    <BreadcrumbList>
                        <BreadcrumbItem>总览</BreadcrumbItem>
                    </BreadcrumbList>
                </Breadcrumb>
            </div>
        </header>
        <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
            <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 py-4">
                <Card v-for="stat in fetchedStatistics" :key="stat.label">
                    <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle class="text-sm font-medium">
                            {{ stat.label }}
                        </CardTitle>
                        <component :is="stat.icon" class="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent class="flex flex-col items-center">
                        <div class="text-2xl font-bold font-mono">
                            {{ stat.total }}
                        </div>
                        <p class="text-xs text-muted-foreground">
                            {{ stat.description }}
                        </p>
                    </CardContent>
                </Card>
            </div>
        </div>
    </SidebarInset>
</template>
