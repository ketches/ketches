<script setup lang="ts">
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { ref } from 'vue';
import * as z from 'zod';

import { updateAppInfo } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import type { appModel } from '@/types/app';
import { toast } from 'vue-sonner';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = ref<appModel>(props.app)

const profileFormSchema = toTypedSchema(z.object({
    slug: z
        .string(
            { required_error: '应用标识是必填项。' }
        )
        .optional(),
    displayName: z
        .string()
        .min(1, {
            message: '应用名称是必填项。',
        }),
    description: z
        .string()
        .optional(),
}))

const { handleSubmit } = useForm({
    validationSchema: profileFormSchema,
    initialValues: {
        slug: app.value.slug,
        displayName: app.value.displayName,
        description: app.value.description || '',
    },
})

const onSubmit = handleSubmit(async (values) => {
    if (values.slug === app.value.slug && values.displayName === app.value.displayName && values.description === app.value.description) {
        toast.warning('没有修改任何内容。')
        return
    }
    await updateAppInfo(app.value.appID, {
        displayName: values.displayName,
        description: values.description ?? '',
    })
    app.value.displayName = values.displayName
    app.value.description = values.description ?? ''
})
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            应用基础信息
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用程序的基础信息，包括应用名称和描述等。
        </p>
    </div>
    <Separator />
    <form class="space-y-8" @submit="onSubmit">
        <FormField v-slot="{ componentField }" name="slug">
            <FormItem>
                <FormLabel>
                    应用标识
                </FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" disabled />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="displayName">
            <FormItem>
                <FormLabel>
                    应用名称
                </FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="description">
            <FormItem>
                <FormLabel>
                    应用描述
                </FormLabel>
                <FormControl>
                    <Textarea type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <div class="flex gap-2 justify-start">
            <Button type="submit">
                保存
            </Button>
        </div>
    </form>
</template>
