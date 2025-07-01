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

import { updateAppInfo } from '@/api/app';
import { useUserStore } from '@/stores/userStore';
import type { appModel, updateAppInfoModel } from '@/types/app';
import { toTypedSchema } from '@vee-validate/zod';
import { Save } from 'lucide-vue-next';
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
    app: {
        type: Object as () => appModel,
        required: false,
    },
})

const emit = defineEmits(['update:modelValue', 'app-updated']);

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
            required_error: '应用标识是必填项。',
        })
        .min(2)
        .max(32),
    displayName: z
        .string({
            required_error: '应用名称是必填项。',
        })
        .min(2)
        .max(100, {
            message: '应用名称必须最多 100 个字符。',
        }),
    description: z
        .string()
        .optional(),
}));

const { isFieldDirty, handleSubmit, values, setFieldValue, resetForm } = useForm({
    validationSchema: formSchema,
})

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                slug: props.app.slug,
                displayName: props.app.displayName,
                description: props.app.description,
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
    const resp = await updateAppInfo(props.app.appID, values as updateAppInfoModel)
    if (resp) {
        userStore.addOrUpdateApp(resp);
        toast.success('更新应用基础信息成功！');
        emit('app-updated');
    }
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>更新应用信息</DialogTitle>
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
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel class="flex items-center gap-1">应用描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full text-2xl max-h-32" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        <Save />
                        保存
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
