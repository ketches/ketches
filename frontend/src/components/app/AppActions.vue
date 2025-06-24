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
import type { appModel } from "@/types/app";
import {
  ArrowUpDown,
  CloudCog,
  CloudUpload,
  MoreHorizontal,
  Play,
  Power,
  Trash
} from "lucide-vue-next";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";

const router = useRouter();

const props = defineProps({
  app: {
    type: Object as () => appModel,
    required: true,
  },
  fromAppList: {
    type: Boolean,
    default: false,
  },
});
const emit = defineEmits(["action-completed"]);

export interface appAction {
  label: string;
  icon: any;
  action: () => void;
}

const appActions = computed<appAction[]>(() => {
  const appStatus = props.app.status;
  switch (appStatus) {
    case "undeployed":
      return [{ label: "部署", icon: CloudCog, action: onDeploy }];
    case "deployed":
    case "stopped":
      return [{ label: "启动", icon: Play, action: onStart }];
    case "starting":
    case "rollingUpdate":
    case "abnormal":
      return [
        { label: "关闭", icon: Power, action: onStop },
        { label: "重新部署", icon: CloudUpload, action: onRedeploy },
      ];
    case "running":
      return [
        { label: "关闭", icon: Power, action: onStop },
        { label: "滚动更新", icon: ArrowUpDown, action: onRollingUpdate },
        { label: "重新部署", icon: CloudUpload, action: onRedeploy },
      ];
    case "completed":
      return [
        { label: "关闭", icon: Power, action: onStop },
        { label: "重新部署", icon: CloudUpload, action: onRedeploy },
      ];
    case "error":
    default:
      return [];
  }
});

async function onDeploy() {
  await appAction(props.app.appID, "deploy").then(() => {
    toast.success("部署成功！", {
      description: `应用 ${props.app.slug} 已成功部署。`,
    });
    emit("action-completed");
  });
}

async function onRedeploy() {
  await appAction(props.app.appID, "redeploy").then(() => {
    toast.success("应用正在重新部署", {
      description: `应用 ${props.app.slug} 正在重新部署。`,
    });
    emit("action-completed");
  });
}

async function onStart() {
  await appAction(props.app.appID, "start").then(() => {
    toast.success("应用正在启动", {
      description: `应用 ${props.app.slug} 正在启动。`,
    });
    emit("action-completed");
  });
}

async function onStop() {
  await appAction(props.app.appID, "stop").then(() => {
    toast.success("应用正在停止", {
      description: `应用 ${props.app.slug} 正在停止。`,
    });
    emit("action-completed");
  });
}

async function onRollingUpdate() {
  await appAction(props.app.appID, "rollingUpdate").then(() => {
    toast.success("应用正在更新", {
      description: `应用 ${props.app.slug} 正在进行滚动更新。`,
    });
    emit("action-completed");
  });
}

const showDeleteAppDialog = ref(false);
async function onDelete() {
  await deleteApp(props.app.appID).then(() => {
    toast.success("应用已删除", {
      description: `应用 ${props.app.slug} 已成功删除。`,
    });
    emit("action-completed");
    if (!props.fromAppList) {
      router.push({ name: "app" });
    }
  });

  showDeleteAppDialog.value = false;
}
</script>

<template>
  <div class="flex items-center gap-2">
    <Button v-for="action in appActions.slice(0, 2)" :key="action.label" @click="action.action" size="sm"
      :variant="fromAppList ? 'ghost' : 'outline'" class="data-[state=open]:bg-muted">
      <component :is="action.icon" class="h-4 w-4" />
      <span v-if="!fromAppList">{{ action.label }}</span>
    </Button>
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" class="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
          <MoreHorizontal class="h-4 w-4" />
          <span class="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem v-for="action in appActions.slice(2)" :key="action.label" @click="action.action">
          <component :is="action.icon" class="mr-2 h-4 w-4" />
          <span>{{ action.label }}</span>
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
</template>
