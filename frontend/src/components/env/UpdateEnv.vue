<script setup lang="ts">
import { getEnv, updateEnv } from '@/api/env';
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
import { useUserStore } from '@/stores/userStore';
import type { envModel, updateEnvModel } from '@/types/env';
import { toTypedSchema } from '@vee-validate/zod';
import { Save } from 'lucide-vue-next';
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
    envID: {
        type: String,
        required: true,
    },
})

const userStore = useUserStore()

const env = ref<envModel>();

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
            required_error: '环境标识必填',
        }),
    displayName: z
        .string({
            required_error: '环境名称必填',
        })
        .min(2, '环境名称最少需要 2 个字符')
        .max(50, {
            message: '环境名称最长不能超过 50 个字符',
        }),
    description: z
        .string()
        .optional()
}));

const { isFieldDirty, handleSubmit, resetForm } = useForm({
    validationSchema: formSchema,
})

watch(open, async (isOpen) => {
    if (isOpen) {
        env.value = await getEnv(props.envID);
        if (env.value) {
            resetForm({
                values: {
                    slug: env.value.slug,
                    displayName: env.value.displayName,
                    description: env.value.description || '',
                },
            });
        }
    }
});

const onSubmit = handleSubmit(async (values) => {
    let envID = env.value?.envID;
    if (!envID) {
        toast.error('环境未提供，请检查配置。');
        return;
    }

    const resp = await updateEnv(envID, {
        displayName: values.displayName,
        description: values.description
    } as updateEnvModel)
    if (resp) {
        toast.success('环境更新成功！');
        userStore.addOrUpdateEnv(resp);
    }

    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>更新环境</DialogTitle>
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
                                    <TooltipTrigger>
                                        环境标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>环境标识用于唯一标识环境。</p>
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
                                        环境名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>环境名称用于展示，便于识别。</p>
                                        <li>长度为 2 到 50 个字符</li>
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
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>环境描述</FormLabel>
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
