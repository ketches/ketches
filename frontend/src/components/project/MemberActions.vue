<script setup lang="ts">
import { removeProjectMembers } from "@/api/project";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { projectMemberModel } from "@/types/project";
import { MoreHorizontal, Trash } from "lucide-vue-next";
import { ref } from "vue";
import { toast } from "vue-sonner";

const emit = defineEmits(["action-completed"]);

const props = defineProps({
    member: {
        type: Object as () => projectMemberModel,
        required: true,
    },
});

const showRemoveMemberDialog = ref(false);
async function onRemove() {
    await removeProjectMembers(props.member.projectID, [
        props.member.userID,
    ]).then(() => {
        toast.success("成员已移除", {
            description: `成员 ${props.member.fullname} 已成功移除。`,
        });
    });

    emit("action-completed");
    showRemoveMemberDialog.value = false;
}

// const openUpdateEnvForm = ref(false)
</script>

<template>
    <DropdownMenu>
        <DropdownMenuTrigger as-child>
            <Button variant="ghost" class="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
                <MoreHorizontal class="h-4 w-4" />
                <span class="sr-only">Open menu</span>
            </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
            <DropdownMenuItem @select.prevent="showRemoveMemberDialog = true"
                class="text-destructive focus:text-destructive">
                <Trash class="mr-2 h-4 w-4" />
                移除
            </DropdownMenuItem>
            <ConfirmDialog v-if="showRemoveMemberDialog" title="移除成员" description="您确定要移除此成员吗？" @confirm="onRemove"
                @cancel="showRemoveMemberDialog = false" />
        </DropdownMenuContent>
    </DropdownMenu>
</template>
