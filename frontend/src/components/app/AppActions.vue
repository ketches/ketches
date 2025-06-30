<script setup lang="ts">
import { appAction, deleteApp } from "@/api/app";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import type { appModel, appRunningInfoModel } from "@/types/app";
import {
  BugPlay,
  Dot,
  MoreVertical,
  Trash
} from "lucide-vue-next";
import { computed, ref, toRef } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../ui/tooltip";
import { appStatusActions, type appStatusAction } from "./data/appStatus";

const router = useRouter();

const props = defineProps({
  appEdition: {
    type: String,
    default: "",
  },
  app: {
    type: Object as () => appModel,
    required: false,
  },
  appRunningInfo: {
    type: Object as () => appRunningInfoModel,
    required: true,
  },
  fromAppList: {
    type: Boolean,
    default: false,
  },
});

const appRunningInfo = toRef(props, 'appRunningInfo');
const appEdition = toRef(props, 'appEdition');
const app = toRef(props, 'app');

const emit = defineEmits(["action-completed"]);

const appActions = computed<appStatusAction[]>(() => {
  if (props.fromAppList) {
    return appStatusActions(app.value.status, appEdition.value || "", app.value.actualEdition) || [];
  }
  return appStatusActions(appRunningInfo.value.status, appEdition.value || "", appRunningInfo.value.actualEdition) || [];
});

const debugActionAvailable = computed(() => {
  if (props.fromAppList) {
    return false; // Debug action not available in app list
  }
  if (["starting", "updating", "abnormal", "running"].includes(appRunningInfo.value.status)) {
    return true; // Debug action available in these statuses
  }
});

const showDeleteAppDialog = ref(false);
const showDebugAppDialog = ref(false);

async function onDelete() {
  await deleteApp(appRunningInfo.value.appID)
  toast.success("应用已删除", {
    description: `应用 ${appRunningInfo.value.slug} 已成功删除。`,
  });
  emit("action-completed");
  if (!props.fromAppList) {
    router.push({ name: "app" });
  }

  showDeleteAppDialog.value = false;
}

async function onDebug() {
  await appAction(appRunningInfo.value.appID, "debug")
  toast.success("应用开始进入调试", {
    description: `等待应用实例重启完成后，您可以进入实例终端进行调试操作。`,
  });
  emit("action-completed");
  showDebugAppDialog.value = false;
}

async function onAction(action: (appID: string) => Promise<appModel> | Promise<void>) {
  await action(appRunningInfo.value.appID)
  emit("action-completed");
}
</script>

<template>
  <div class="flex items-center gap-2">
    <TooltipProvider v-for="action in appActions?.slice(0, 2)" :delay-duration="500" :disabled="!fromAppList">
      <Tooltip>
        <TooltipTrigger as-child>
          <Button :key="action.label" @click="onAction(action.action)" size="sm"
            :variant="fromAppList ? 'ghost' : 'outline'" class="data-[state=open]:bg-muted font-normal">
            <component :is="action.icon" :class="`${action.tip ? 'text-blue-500' : ''} h-4 w-4`" />
            <Dot v-if="action.tip" class="text-blue-500" stroke-width="8" />
            <span v-if="!fromAppList">{{ action.label }}</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>{{ action.label }}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button :variant="fromAppList ? 'ghost' : 'outline'" class="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
          <MoreVertical class="h-4 w-4" />
          <span class="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem v-for="action in appActions?.slice(2)" :key="action.label" @click="onAction(action.action)">
          <component :is="action.icon" class="mr-2 h-4 w-4" />
          <Dot v-if="action.tip" class="text-blue-500" stroke-width="8" />
          <span>{{ action.label }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem v-if="debugActionAvailable" @click.prevent="showDebugAppDialog = true"
          class="text-orange-500 focus:text-orange-500">
          <BugPlay class="mr-2 h-4 w-4 text-orange-500" />
          调试
        </DropdownMenuItem>
        <DropdownMenuSeparator v-if="appActions.length > 2" />
        <DropdownMenuItem @select.prevent="showDeleteAppDialog = true" class="text-destructive focus:text-destructive">
          <Trash class="mr-2 h-4 w-4 text-destructive" />
          删除
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
  <ConfirmDialog v-if="showDeleteAppDialog" title="删除应用" description="您确定要删除此应用吗？此操作无法撤销。" @confirm="onDelete"
    @cancel="showDeleteAppDialog = false" />
  <ConfirmDialog v-if="showDebugAppDialog" title="调试应用" description="调试期间应用服务不可用，确定要继续吗？" @confirm="onDebug"
    @cancel="showDebugAppDialog = false" />
</template>
