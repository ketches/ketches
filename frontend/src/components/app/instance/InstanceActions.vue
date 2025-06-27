<script setup lang="ts">
import { terminateAppInstance } from "@/api/app";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import ContainerLogsDrawer from "@/components/shared/ContainerLogsDrawer.vue";
import ContainerTerminalDrawer from "@/components/shared/ContainerTerminalDrawer.vue";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';
import type { appInstanceModel } from "@/types/app";
import { Logs, MoreVertical, SquareX, Terminal, View } from "lucide-vue-next";
import { ref } from "vue";
import { toast } from "vue-sonner";

const emit = defineEmits([
    "action-completed",
    "view-instance",
    "view-logs",
    "view-console",
]);

const props = defineProps({
    appInstance: {
        type: Object as () => appInstanceModel,
        required: true,
    },
});

const showTerminateInstanceDialog = ref(false);

async function onTerminate() {
    await terminateAppInstance(
        props.appInstance.appID,
        props.appInstance.instanceName
    ).then(() => {
        toast.info("已发送终止请求", {
            description: `实例 ${props.appInstance.instanceName} 的终止请求已发送。`,
        });
    });

    emit("action-completed");
    showTerminateInstanceDialog.value = false;
}

const openContainerLogsDrawer = ref(false);
const openContainerTerminalDrawer = ref(false);
</script>

<template>
    <div class="flex items-center gap-2">
        <TooltipProvider :delay-duration="500">
            <Tooltip>
                <TooltipTrigger as-child>
                    <Button variant="ghost" class="h-8 w-8 p-0" @click="$emit('view-instance')">
                        <View class="h-4 w-4" />
                        <span class="sr-only">查看</span>
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>查看实例详情</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
        <TooltipProvider :delay-duration="500">
            <Tooltip>
                <TooltipTrigger as-child>
                    <Button variant="ghost" class="h-8 w-8 p-0" @click="openContainerLogsDrawer = true">
                        <Logs class="h-4 w-4" />
                        <span class="sr-only">日志</span>
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>查看日志</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
        <TooltipProvider :delay-duration="500">
            <Tooltip>
                <TooltipTrigger as-child>
                    <Button variant="ghost" class="h-8 w-8 p-0" @click="openContainerTerminalDrawer = true">
                        <Terminal class="h-4 w-4" />
                        <span class="sr-only">终端</span>
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>容器终端</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
        <DropdownMenu>
            <DropdownMenuTrigger as-child>
                <Button variant="ghost" class="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
                    <MoreVertical class="h-4 w-4" />
                    <span class="sr-only">Open menu</span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                <DropdownMenuItem @select.prevent="showTerminateInstanceDialog = true"
                    class="text-destructive focus:text-destructive">
                    <SquareX class="mr-2 h-4 w-4 text-destructive" />
                    终止
                </DropdownMenuItem>
                <ConfirmDialog v-if="showTerminateInstanceDialog" title="终止实例" description="您确定要终止此实例吗？"
                    @confirm="onTerminate" @cancel="showTerminateInstanceDialog = false" />
            </DropdownMenuContent>
        </DropdownMenu>
    </div>
    <ContainerLogsDrawer v-model="openContainerLogsDrawer" :appID="props.appInstance.appID"
        :instanceName="props.appInstance.instanceName"
        :containers="(appInstance.containers || []).map(c => c.containerName)"
        :initContainers="(appInstance.initContainers || []).map(c => c.containerName)" :showOptions="true" />
    <ContainerTerminalDrawer v-model="openContainerTerminalDrawer" :appID="props.appInstance.appID"
        :instanceName="props.appInstance.instanceName"
        :containers="(appInstance.containers || []).map(c => c.containerName)"
        :initContainers="(appInstance.initContainers || []).map(c => c.containerName)" />
</template>
