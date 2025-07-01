<script setup lang="ts">
import { getCluster, updateCluster } from '@/api/cluster';
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
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';
import type { clusterModel, updateClusterModel } from '@/types/cluster';
import { toTypedSchema } from '@vee-validate/zod';
import { CloudUpload, Link, Save } from 'lucide-vue-next';
import { useForm } from 'vee-validate';
import { computed, ref, watch } from 'vue';
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
    clusterID: {
        type: String,
        required: true,
    },
})


const cluster = ref<clusterModel>();

const emit = defineEmits(['update:modelValue', 'cluster-updated']);

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
        }),
    displayName: z
        .string({
            required_error: '集群名称必填',
        })
        .min(2, '集群名称最少需要 2 个字符')
        .max(50, {
            message: '集群名称最长不能超过 50 个字符'
        }),
    kubeConfig: z
        .string({
            required_error: 'KubeConfig 必填'
        }).min(1, 'KubeConfig 必填'),
    description: z
        .string()
        .optional(),
}));

const { isFieldDirty, handleSubmit, resetForm, setFieldValue } = useForm({
    validationSchema: formSchema,
})

watch(open, async (isOpen) => {
    if (isOpen) {
        cluster.value = await getCluster(props.clusterID);
        if (cluster.value) {
            resetForm({
                values: {
                    slug: cluster.value.slug,
                    displayName: cluster.value.displayName,
                    kubeConfig: cluster.value.kubeConfig || '',
                    description: cluster.value.description || '',
                },
            });
        }
    }
});

const onSubmit = handleSubmit(async (values) => {
    let clusterID = cluster.value?.clusterID;
    if (!clusterID) {
        toast.error('集群未提供，请检查配置。');
        return;
    }

    const resp = await updateCluster(clusterID, values as updateClusterModel)
    if (resp) {
        toast.success('环境更新成功！');
        emit('cluster-updated');
    }

    open.value = false;
})

const fileInputRef = ref<HTMLInputElement | null>(null);

function handleUploadClick() {
    fileInputRef.value?.click();
}

function handleFileChange(e: Event) {
    const files = (e.target as HTMLInputElement).files;
    if (files && files.length > 0) {
        const file = files[0];
        const reader = new FileReader();
        reader.onload = (event) => {
            setFieldValue('kubeConfig', event.target?.result as string || '');
        };
        reader.readAsText(file);
    }
}
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[700px]">
            <DialogHeader>
                <DialogTitle>更新集群</DialogTitle>
                <DialogDescription>
                    填写集群的标识和名称，标识用于唯一标识集群，名称用于展示。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        集群标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>集群标识用于唯一标识集群。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" disabled />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="displayName" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        集群名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>集群名称用于展示，便于识别。</p>
                                        <li>长度为 2 到 50 个字符</li>
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
                        <FormLabel class="flex items-center gap-2">
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        KubeConfig
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>KubeConfig 是集群的配置文件，用于连接和管理集群。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                            <Button variant="link" size="sm"
                                class="h-4 text-xs text-muted-foreground ml-auto hover:text-primary" type="button"
                                @click="handleUploadClick">
                                <CloudUpload />
                                上传 KubeConfig 配置文件
                            </Button>
                            <input ref="fileInputRef" type="file" accept="" class="hidden" @change="handleFileChange" />
                        </FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full bg-accent font-mono text-xs max-h-32"
                                placeholder="" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>集群描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full text-2xl max-h-32" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <DialogFooter class="flex w-full px-0">
                    <Button v-if="cluster?.kubeConfig" variant="outline" type="button" @click="open = false"
                        class="mr-auto">
                        <Link />
                        连通性测试
                    </Button>
                    <Button type="submit" class="ml-auto min-w-[100px]">
                        <Save />
                        保存
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
