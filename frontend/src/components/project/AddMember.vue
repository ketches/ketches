<script setup lang="ts">
import { addProjectMember, listAddableProjectMembers } from '@/api/project';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle
} from '@/components/ui/dialog';
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form';
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@/components/ui/select';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';
import { useUserStore } from '@/stores/userStore';
import type { userRefModel } from '@/types/user';
import { toTypedSchema } from '@vee-validate/zod';
import { storeToRefs } from 'pinia';
import { useForm } from 'vee-validate';
import { computed, ref, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import Button from '../ui/button/Button.vue';
import DialogFooter from '../ui/dialog/DialogFooter.vue';
import { projectRoleRefs } from './data/projectRole';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
})

const userStore = useUserStore()
const { activeProjectRef } = storeToRefs(userStore)

const emit = defineEmits(['update:modelValue', 'member-added']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const addableMembers = ref<userRefModel[]>([]);

watch(open, async (isOpen) => {
    if (isOpen) {
        addableMembers.value = await listAddableProjectMembers(activeProjectRef.value?.projectID || '');
    }
});

const formSchema = toTypedSchema(z.object({
    userIDs: z
        .array(z.string())
        .min(1, {
            message: '至少需要添加一个成员。',
        }),
    projectRole: z
        .string({
            required_error: '角色是必填项。',
        })
}));

const { isFieldDirty, handleSubmit } = useForm({
    validationSchema: formSchema,
})

const onSubmit = handleSubmit(async (values) => {
    if (!activeProjectRef.value) {
        toast.error('未找到当前项目，请先选择一个项目。');
    } else {
        const resp = await addProjectMember(activeProjectRef.value.projectID, values.userIDs, values.projectRole)
        if (resp) {
            emit('member-added');
            toast.success('添加成员成功！');
        }
    }
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>添加成员</DialogTitle>
                <DialogDescription>
                    添加成员到当前项目中，支持一次添加多个成员。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="userIDs" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        成员
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        支持一次添加多个成员。
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <!-- <Input v-bind="componentField" class="w-full" /> -->
                            <Select v-bind="componentField" multiple>
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择成员" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem v-for="user in addableMembers" :key="user.userID"
                                            :value="user.userID">
                                            {{ user.fullname }}
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="projectRole" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        角色
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>成员区分不同角色，细化权限管理。例如：</p>
                                        <li>所有者：拥有最高权限，可以管理项目中的所有资源。</li>
                                        <li>开发者：可以管理应用和资源，但不能管理项目。</li>
                                        <li>观察者：只能查看资源，无编辑或管理权限。</li>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <!-- <TagsInput v-model="modelValue"> -->
                        <!-- <TagsInput v-model="modelValue" v-bind="componentField">
                            <TagsInputItem v-for="user in addableMembers" :key="user.userID" :value="user.userID">
                                <TagsInputItemText />
                                <TagsInputItemDelete />
                            </TagsInputItem>

                            <TagsInputInput placeholder="Fruits..." />
                        </TagsInput> -->
                        <Select v-bind="componentField">
                            <FormControl>
                                <SelectTrigger class="w-full">
                                    <SelectValue>
                                        <div v-if="componentField.modelValue" class="flex items-center">
                                            <component
                                                :is="projectRoleRefs[componentField.modelValue as keyof typeof projectRoleRefs]?.icon"
                                                class="h-4 w-4 mr-2" />
                                            <span>{{
                                                projectRoleRefs[componentField.modelValue as keyof typeof
                                                    projectRoleRefs]?.label
                                            }}</span>
                                        </div>
                                        <span v-else>选择角色</span>
                                    </SelectValue>
                                </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                                <SelectGroup>
                                    <SelectItem v-for="(role, key) in projectRoleRefs" :key="key" :value="key">
                                        <div class="flex items-center">
                                            <component :is="role.icon" class="h-4 w-4 mr-2" />
                                            <span>{{ role.label }}</span>
                                        </div>
                                    </SelectItem>
                                </SelectGroup>
                            </SelectContent>
                        </Select>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        添加成员
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
