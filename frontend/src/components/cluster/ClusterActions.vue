<script setup lang="ts">
import { deleteCluster } from "@/api/cluster";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { clusterModel } from "@/types/cluster";
import { Edit, MoreVertical, Trash } from "lucide-vue-next";
import { ref } from "vue";
import { toast } from "vue-sonner";
import UpdateCluster from "./UpdateCluster.vue";

const emit = defineEmits(["action-completed"]);

const props = defineProps({
  cluster: {
    type: Object as () => clusterModel,
    required: true,
  },
});

const showDeleteClusterDialog = ref(false);
async function onDelete() {
  await deleteCluster(props.cluster.clusterID).then(() => {
    toast.success("集群已删除", {
      description: `集群 ${props.cluster.slug} 已成功删除。`,
    });
  });

  emit("action-completed");
  showDeleteClusterDialog.value = false;
}

const openUpdateClusterForm = ref(false);
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
      <DropdownMenuItem @select.prevent="openUpdateClusterForm = true">
        <Edit class="mr-2 h-4 w-4" />
        编辑
      </DropdownMenuItem>
      <DropdownMenuItem @select.prevent="showDeleteClusterDialog = true"
        class="text-destructive focus:text-destructive">
        <Trash class="text-destructive mr-2 h-4 w-4" />
        删除
      </DropdownMenuItem>
      <ConfirmDialog v-if="showDeleteClusterDialog" title="删除集群" description="您确定要删除此集群吗？此操作无法撤销。" @confirm="onDelete"
        @cancel="showDeleteClusterDialog = false" />
    </DropdownMenuContent>
  </DropdownMenu>
  <UpdateCluster v-model="openUpdateClusterForm" :clusterID="props.cluster.clusterID"
    @cluster-updated="emit('action-completed')" />
</template>
