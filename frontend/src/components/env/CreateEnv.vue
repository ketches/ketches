<script setup lang="ts">
import { createEnv } from '@/api/project';
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
import { useClusterStore } from '@/stores/clusterStore';
import { useUserStore } from '@/stores/userStore';
import type { envCreateModel } from '@/types/env';
import { toTypedSchema } from '@vee-validate/zod';
import { storeToRefs } from 'pinia';
import { useForm } from 'vee-validate';
import { computed, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import Button from '../ui/button/Button.vue';
import DialogFooter from '../ui/dialog/DialogFooter.vue';
import Input from '../ui/input/Input.vue';
import Textarea from '../ui/textarea/Textarea.vue';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
})

const userStore = useUserStore()
const { activeProjectRef } = storeToRefs(userStore)

const emit = defineEmits(['update:modelValue', 'env-created']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const clusterStore = useClusterStore();

watch(open, async (isOpen) => {
    if (isOpen) {
        await useClusterStore().loadClusterRefs();
    }
});

const formSchema = toTypedSchema(z.object({
    slug: z
        .string({
            required_error: 'Slug is required.',
        })
        .min(2)
        .max(32),
    displayName: z
        .string({
            required_error: 'Display name is required.',
        })
        .min(2)
        .max(100, {
            message: 'Display name must be at most 100 characters long.',
        }),
    clusterID: z
        .string({
            required_error: 'Cluster is required.',
        }),
    description: z
        .string()
        .optional(),
}));

const { isFieldDirty, handleSubmit } = useForm({
    validationSchema: formSchema,
})

const onSubmit = handleSubmit(async (values) => {
    if (!activeProjectRef.value) {
        toast.error('未找到当前项目，请先选择一个项目。');
    } else {
        const resp = await createEnv(activeProjectRef.value.projectID, values as envCreateModel)
        if (resp) {
            userStore.addOrUpdateEnv({
                envID: resp.envID,
                slug: resp.slug,
                displayName: resp.displayName,
                projectID: resp.projectID,
            });
            toast.success('创建环境成功！');
            emit('env-created');
        }
    }
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>创建环境</DialogTitle>
                <DialogDescription>
                    填写环境的标识和名称，标识用于唯一标识环境，名称用于展示。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        环境标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>环境标识用于唯一标识环境，不能重复。</p>
                                        <li>只能包含小写字母、数字和短横线</li>
                                        <li>必须以字母开头</li>
                                        <li>不能以短横线结尾。</li>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="displayName" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        环境名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>环境名称用于展示，便于识别。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" placeholder="例如：开发环境、预发布环境、生产环境" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="clusterID" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        集群
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        选择环境所属的集群。
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <Select v-bind="componentField">
                            <FormControl>
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择集群" />
                                </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                                <SelectGroup>
                                    <SelectItem v-for="clusterRef in clusterStore.clusterRefs"
                                        :key="clusterRef.clusterID" :value="clusterRef.clusterID">
                                        {{ clusterRef.displayName }}
                                    </SelectItem>
                                </SelectGroup>
                            </SelectContent>
                        </Select>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>环境描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="col-span-3" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        创建
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
