<script setup lang="ts">
import { deleteEnv } from "@/api/env";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useResourceRefStore } from "@/stores/resourceRefStore";
import type { envModel } from "@/types/env";
import { Edit, MoreVertical, Trash } from "lucide-vue-next";
import { ref } from "vue";
import { toast } from "vue-sonner";
import UpdateEnv from "./UpdateEnv.vue";

const emit = defineEmits(["action-completed"]);

const resourceRefStore = useResourceRefStore();

const props = defineProps({
  env: {
    type: Object as () => envModel,
    required: true,
  },
});

const showDeleteEnvDialog = ref(false);
async function onDelete() {
  await deleteEnv(props.env.envID).then(() => {
    toast.success("环境已删除", {
      description: `环境 ${props.env.slug} 已成功删除。`,
    });
    resourceRefStore.removeEnv(props.env.envID);
  });

  emit("action-completed");
  showDeleteEnvDialog.value = false;
}

const openUpdateEnvForm = ref(false);
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
        <MoreVertical class="h-4 w-4" />
        <span class="sr-only">Open menu</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem @select.prevent="openUpdateEnvForm = true">
        <Edit class="mr-2 h-4 w-4" />
        编辑
      </DropdownMenuItem>
      <DropdownMenuItem @select.prevent="showDeleteEnvDialog = true" class="text-destructive focus:text-destructive">
        <Trash class="text-destructive mr-2 h-4 w-4" />
        删除
      </DropdownMenuItem>
      <ConfirmDialog v-if="showDeleteEnvDialog" title="删除环境" description="您确定要删除此环境吗？此操作无法撤销。" @confirm="onDelete"
        @cancel="showDeleteEnvDialog = false" />
    </DropdownMenuContent>
  </DropdownMenu>
  <UpdateEnv v-model="openUpdateEnvForm" :envID="props.env.envID" />
</template>
