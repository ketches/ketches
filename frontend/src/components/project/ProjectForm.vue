<script setup lang="ts">
import { createProject } from '@/api/project';
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
import { useUserStore } from '@/stores/userStore';
import type { createProjectModel } from '@/types/project';
import { toTypedSchema } from '@vee-validate/zod';
import { Plus } from 'lucide-vue-next';
import { useForm } from 'vee-validate';
import { computed } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import Button from '../ui/button/Button.vue';
import Textarea from '../ui/textarea/Textarea.vue';

const userStore = useUserStore();

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
            required_error: '项目标识必填',
        })
        .min(3, '长度不能小于 3')
        .max(32, '长度不能大于 32')
        .regex(/^[a-z]/, '必须以小写字母开头')
        .regex(/[a-z0-9]$/, '不能以短横线结尾')
        .regex(/^[a-z0-9-]+$/, '只能包含小写字母、数字和短横线'),
    displayName: z
        .string({
            required_error: '项目名称必填',
        })
        .min(2, '项目名称最少需要 2 个字符')
        .max(50, {
            message: '项目名称最长不能超过 50 个字符',
        }),
    description: z
        .string()
        .optional(),
}));

const { isFieldDirty, handleSubmit } = useForm({
    validationSchema: formSchema,
})

const onSubmit = handleSubmit(async (values) => {
    const resp = await createProject(values as createProjectModel)
    if (resp) {
        userStore.addOrUpdateProject({
            projectID: resp.projectID,
            slug: resp.slug,
            displayName: resp.displayName,
        });
        toast.success('创建项目成功！');
    } else {
        toast.error('创建项目失败，请重试。');
    }

    open.value = false;
})

</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>创建项目</DialogTitle>
                <DialogDescription>
                    填写项目的标识和名称，标识用于唯一标识项目，名称用于展示。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger>
                                        项目标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>项目标识用于唯一标识项目，不能重复。</p>
                                        <li>只能包含小写字母、数字和短横线</li>
                                        <li>必须以字母开头，不能以短横线结尾</li>
                                        <li>长度为 3 到 32 个字符</li>
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
                                    <TooltipTrigger>
                                        项目名称
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>项目名称用于展示，便于识别。</p>
                                        <li>长度为 2 到 50 个字符</li>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" placeholder="例如：我的项目、测试项目" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="description" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>项目描述</FormLabel>
                        <FormControl>
                            <Textarea v-bind="componentField" class="w-full text-2xl max-h-32" placeholder="项目的详细描述" />
                        </FormControl>
                        <FormMessage />
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
