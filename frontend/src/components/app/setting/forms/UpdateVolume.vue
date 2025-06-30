<script setup lang="ts">
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

import { updateAppVolume } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import type { appVolumeModel } from '@/types/app';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { computed, ref, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
    volume: {
        type: Object as () => appVolumeModel,
        required: true,
    },
})

const emit = defineEmits(['update:modelValue', 'volume-updated']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const formSchema = toTypedSchema(z.object({
    slug: z
        .string()
        .min(1, {
            message: '存储标识是必填项。',
        }).max(32, {
            message: '存储标识不能超过32个字符。',
        })
        .max(32),
    mountPath: z
        .string()
        .min(1, {
            message: '挂载路径是必填项。',
        }),
    subPath: z.string().optional(),
}));

const { isFieldDirty, handleSubmit, resetForm } = useForm({
    validationSchema: formSchema,
})

const subPathHasValue = ref(false);


// 表单初始化时同步 subPath
watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                slug: props.volume.slug,
                mountPath: props.volume.mountPath,
                subPath: props.volume.subPath || '',
            }
        });
        subPathHasValue.value = !!props.volume.subPath;
    }
}, { immediate: true })

// 提交时同步 subPath 到表单
const onSubmit = handleSubmit(async (values) => {
    await updateAppVolume(props.volume.appID, props.volume.volumeID, {
        // slug: values.slug,
        mountPath: values.mountPath,
        subPath: values.subPath,
    });
    toast.success('存储卷更新成功！');
    emit('volume-updated');
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>更新应用存储卷</DialogTitle>
                <DialogDescription>
                    请填写应用存储卷的相关信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储卷标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>存储卷标识用于唯一标识存储卷，不能重复。</p>
                                        <li>只能包含小写字母、数字和短横线</li>
                                        <li>必须以字母开头</li>
                                        <li>不能以短横线结尾。</li>
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
                <FormField v-slot="{ componentField }" name="mountPath" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        挂载路径
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>挂载路径用于指定存储卷在容器内的挂载位置。</p>
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
                <p class="text-sm text-muted-foreground flex items-center gap-2">
                    <Checkbox v-model="subPathHasValue" />
                    <label
                        class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                        需要挂载子路径
                    </label>
                </p>
                <FormField v-if="subPathHasValue" v-slot="{ componentField }" name="subPath"
                    :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储到子路径
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>用于指定容器挂载存储卷的子路径。</p>
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
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        创建
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
