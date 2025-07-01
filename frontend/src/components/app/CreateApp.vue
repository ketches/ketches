<script setup lang="ts">
import { createApp } from '@/api/env';
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
import type { createAppModel } from '@/types/app';
import { toTypedSchema } from '@vee-validate/zod';
import { Plus, SquareDashed, SquareDot, SquaresUnite } from 'lucide-vue-next';
import { storeToRefs } from 'pinia';
import { useForm } from 'vee-validate';
import { computed, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import Button from '../ui/button/Button.vue';
import Checkbox from '../ui/checkbox/Checkbox.vue';
import DialogFooter from '../ui/dialog/DialogFooter.vue';
import Input from '../ui/input/Input.vue';
import Textarea from '../ui/textarea/Textarea.vue';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    }
})

const emit = defineEmits(['update:modelValue', 'app-created']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const userStore = useUserStore()
const { activeEnvRef } = storeToRefs(userStore)

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
    workloadType: z
        .string({
            required_error: 'Workload type is required.',
        }).default('Deployment'),
    containerImage: z
        .string({
            required_error: 'Container image is required.',
        }),

    replicas: z.number().min(1, {
        message: 'Replicas must be at least 1.',
    }).default(1),
    description: z
        .string()
        .optional(),
    deploy: z.boolean(),
}));

const { isFieldDirty, handleSubmit, values, setFieldValue, resetForm } = useForm({
    validationSchema: formSchema,
})

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                slug: 'my-app',
                displayName: 'my-app',
                deploy: true,
                workloadType: 'Deployment',
                replicas: 1,
            }
        });
    }
})

watch(() => values.slug, (newSlug, oldSlug) => {
    if (values.displayName === oldSlug || !values.displayName) {
        setFieldValue('displayName', newSlug);
    }
});

const onSubmit = handleSubmit(async (values) => {
    if (!activeEnvRef.value) {
        toast.error('未找到当前环境，请先选择一个环境。');
    } else {
        const resp = await createApp(activeEnvRef.value.envID, values as createAppModel)
        if (resp) {
            userStore.addOrUpdateApp(resp);
            toast.success('创建应用成功！');
            emit('app-created');
        }
    }
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>创建应用</DialogTitle>
                <DialogDescription>
                    请填写应用的标识、名称和工作负载类型等信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        应用标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>应用标识用于唯一标识应用，不能重复。</p>
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
                                        应用名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>应用名称用于展示，便于识别。</p>
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
                <FormField v-slot="{ componentField }" name="workloadType" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        工作负载类型
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>选择应用的工作负载类型，默认为 Deployment。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Select v-bind="componentField" :default-value="'Deployment'">
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择工作负载类型" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem value="Deployment">
                                            <SquareDashed />
                                            Deployment
                                        </SelectItem>
                                        <SelectItem value="StatefulSet">
                                            <SquareDot />
                                            StatefulSet
                                        </SelectItem>
                                        <SelectItem value="DaemonSet" disabled>
                                            <SquaresUnite />
                                            DaemonSet
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="containerImage" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        容器镜像
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>应用的容器镜像地址，例如：nginx:latest</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" placeholder="例如：nginx:latest" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="replicas" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        实例数
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>应用的实例数，至少为1。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" type="number" :default-value="1" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel class="flex items-center gap-1">应用描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full text-2xl max-h-32" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="deploy" :validate-on-blur="!isFieldDirty">
                    <FormItem class="flex items-center gap-2">
                        <FormControl>
                            <Checkbox v-bind="componentField" />
                            <label for="terms"
                                class="text-sm text-muted-foreground leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                                创建并部署应用
                            </label>
                        </FormControl>
                    </FormItem>
                </FormField>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        <Plus />
                        创建
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
