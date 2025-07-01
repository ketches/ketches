<script setup lang="ts">
import { updateProjectMemberRole } from '@/api/project'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { type projectMemberModel } from '@/types/project'
import { Check, SquarePen, X } from 'lucide-vue-next'
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { projectRoleRefs } from './data/projectRole'

const props = defineProps<{
    member: projectMemberModel
    activeProjectID: string
}>()

const emit = defineEmits(['role-updated'])

const isEditing = ref(false)
const selectedRole = ref(props.member.projectRole)

async function handleUpdateRole() {
    await updateProjectMemberRole(props.activeProjectID, props.member.userID, selectedRole.value)
    toast.success('成员角色已更新')
    isEditing.value = false
    emit('role-updated')
}

function handleEditClick() {
    selectedRole.value = props.member.projectRole
    isEditing.value = true
}

</script>

<template>
    <div v-if="isEditing" class="flex items-center justify-center gap-2">
        <Select v-model="selectedRole">
            <SelectTrigger class="w-fit text-xs">
                <SelectValue>
                    <div class="flex items-center">
                        <component :is="projectRoleRefs[selectedRole as keyof typeof projectRoleRefs]?.icon"
                            class="h-4 w-4 mr-2" />
                        <span>{{ projectRoleRefs[selectedRole as keyof typeof projectRoleRefs]?.label || '选择角色'
                        }}</span>
                    </div>
                </SelectValue>
            </SelectTrigger>
            <SelectContent>
                <SelectItem v-for="(role, key) in projectRoleRefs" :key="key" :value="key">
                    <div class="flex items-center">
                        <component :is="role.icon" class="h-4 w-4 mr-2" />
                        <span>{{ role.label }}</span>
                    </div>
                </SelectItem>
            </SelectContent>
        </Select>
        <Button variant="ghost" size="icon" class="h-7 w-7 text-green-500" @click="handleUpdateRole">
            <Check class="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" class="h-7 w-7 text-red-500" @click="isEditing = false">
            <X class="h-4 w-4" />
        </Button>
    </div>
    <div v-else class="flex items-center justify-center">
        <Badge variant="secondary"
            :class="['capitalize flex justify-center text-center', projectRoleRefs[member.projectRole as keyof typeof projectRoleRefs]?.style || 'text-gray-500']">
            <component :is="projectRoleRefs[member.projectRole as keyof typeof projectRoleRefs]?.icon"
                class="h-4 w-4 mr-1" />
            <span>{{ projectRoleRefs[member.projectRole as keyof typeof projectRoleRefs]?.label || member.projectRole
            }}</span>
        </Badge>
        <SquarePen class="ml-2 h-4 w-4 text-xs text-muted-foreground hover:text-primary cursor-pointer" title="编辑角色"
            @click="handleEditClick" />
    </div>
</template>
