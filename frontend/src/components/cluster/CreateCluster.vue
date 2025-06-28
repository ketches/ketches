<script setup lang="ts">
import { createCluster } from '@/api/cluster';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
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
import { Input } from '@/components/ui/input';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';
import type { createClusterModel } from '@/types/cluster';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { computed } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import Button from '../ui/button/Button.vue';
import Textarea from '../ui/textarea/Textarea.vue';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
})

const emit = defineEmits(['update:modelValue']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const formSchema = toTypedSchema(z.object({
    slug: z
        .string({
            required_error: '集群标识必填',
        })
        .min(2)
        .max(32),
    displayName: z
        .string({
            required_error: '集群名称必填',
        })
        .min(2)
        .max(50, {
            message: '集群名称最长不能超过 50'
        }),
    kubeConfig: z
        .string({
            message: 'KubeConfig 必填'
        }),
    description: z
        .string()
        .optional(),
}));

const { isFieldDirty, handleSubmit } = useForm({
    validationSchema: formSchema,
})

const onSubmit = handleSubmit(async (values) => {
    const resp = await createCluster(values as createClusterModel)
    if (resp) {
        toast.success('创建集群成功！');
    } else {
        toast.error('创建集群失败，请重试。');
    }

    open.value = false;
})

</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>创建集群</DialogTitle>
                <DialogDescription>
                    请填写集群的标识、名称和 KubeConfig 等信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        集群标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>集群标识用于唯一标识集群，不能重复。</p>
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
                                        集群名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>集群名称用于展示，便于识别。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" placeholder="" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="kubeConfig" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        KubeConfig
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>KubeConfig 是集群的配置文件，用于连接和管理集群。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full bg-accent font-mono text-xs" placeholder=""
                                rows="8" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>集群描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full" placeholder="" />
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
