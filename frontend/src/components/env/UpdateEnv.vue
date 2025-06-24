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
import { useResourceRefStore } from '@/stores/resourceRefStore';
import type { envModel } from '@/types/env';
import { toTypedSchema } from '@vee-validate/zod';
import { storeToRefs } from 'pinia';
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

const resourceRefStore = useResourceRefStore()
const { envRefs } = storeToRefs(resourceRefStore)

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

    const resp = await updateEnv(envID, values)
    if (resp) {
        toast.success('环境更新成功！');
        envRefs.value = envRefs.value.map(e => e.envID === envID ? resp : e);
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
                        更新
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
